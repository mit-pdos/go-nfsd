package goose_nfs

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"sync"

	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/fh"
	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/nfstypes"

	"testing"

	"github.com/stretchr/testify/assert"
)

var quiet = flag.Bool("quiet", false, "disable logging")

func checkFlags() {
	if *quiet {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}
}

type TestState struct {
	t   *testing.T
	nfs *Nfs
}

func (ts *TestState) CreateOp(fh nfstypes.Nfs_fh3, name string) nfstypes.CREATE3res {
	where := nfstypes.Diropargs3{Dir: fh, Name: nfstypes.Filename3(name)}
	how := nfstypes.Createhow3{}
	args := nfstypes.CREATE3args{Where: where, How: how}
	attr := ts.nfs.NFSPROC3_CREATE(args)
	return attr
}

func (ts *TestState) CreateFh(fh nfstypes.Nfs_fh3, name string) {
	attr := ts.CreateOp(fh, name)
	assert.Equal(ts.t, nfstypes.NFS3_OK, attr.Status)
}

func (ts *TestState) Create(name string) {
	ts.CreateFh(fh.MkRootFh3(), name)
}

func (ts *TestState) LookupOp(fh nfstypes.Nfs_fh3, name string) *nfstypes.LOOKUP3res {
	what := nfstypes.Diropargs3{Dir: fh, Name: nfstypes.Filename3(name)}
	args := nfstypes.LOOKUP3args{What: what}
	reply := ts.nfs.NFSPROC3_LOOKUP(args)
	return &reply
}

func (ts *TestState) LookupFh(fh nfstypes.Nfs_fh3, name string) nfstypes.Nfs_fh3 {
	reply := ts.LookupOp(fh, name)
	assert.Equal(ts.t, reply.Status, nfstypes.NFS3_OK)
	return reply.Resok.Object
}

func (ts *TestState) Lookup(name string, succeed bool) nfstypes.Nfs_fh3 {
	reply := ts.LookupOp(fh.MkRootFh3(), name)
	if succeed {
		assert.Equal(ts.t, reply.Status, nfstypes.NFS3_OK)
	} else {
		assert.NotEqual(ts.t, reply.Status, nfstypes.NFS3_OK)
	}
	return reply.Resok.Object
}

func (ts *TestState) GetattrOp(fh nfstypes.Nfs_fh3) *nfstypes.GETATTR3res {
	args := nfstypes.GETATTR3args{Object: fh}
	attr := ts.nfs.NFSPROC3_GETATTR(args)
	return &attr
}

func (ts *TestState) Getattr(fh nfstypes.Nfs_fh3, sz uint64) {
	attr := ts.GetattrOp(fh)
	assert.Equal(ts.t, nfstypes.NFS3_OK, attr.Status)
	assert.Equal(ts.t, nfstypes.NF3REG, attr.Resok.Obj_attributes.Ftype)
	assert.Equal(ts.t, nfstypes.Size3(sz), attr.Resok.Obj_attributes.Size)
}

func (ts *TestState) GetattrDir(fh nfstypes.Nfs_fh3) {
	attr := ts.GetattrOp(fh)
	assert.Equal(ts.t, nfstypes.NFS3_OK, attr.Status)
	assert.Equal(ts.t, attr.Resok.Obj_attributes.Ftype, nfstypes.NF3DIR)
}

func (ts *TestState) GetattrFail(fh nfstypes.Nfs_fh3) {
	attr := ts.GetattrOp(fh)
	assert.Equal(ts.t, attr.Status, nfstypes.NFS3ERR_STALE)
}

func (ts *TestState) Setattr(fh nfstypes.Nfs_fh3, sz uint64) {
	size := nfstypes.Set_size3{Set_it: true, Size: nfstypes.Size3(sz)}
	attr := nfstypes.Sattr3{Size: size}
	args := nfstypes.SETATTR3args{Object: fh, New_attributes: attr}
	reply := ts.nfs.NFSPROC3_SETATTR(args)
	assert.Equal(ts.t, reply.Status, nfstypes.NFS3_OK)
}

func (ts *TestState) WriteOp(fh nfstypes.Nfs_fh3, off uint64, data []byte, how nfstypes.Stable_how) *nfstypes.WRITE3res {
	args := nfstypes.WRITE3args{
		File:   fh,
		Offset: nfstypes.Offset3(off),
		Count:  nfstypes.Count3(len(data)),
		Stable: how,
		Data:   data}
	reply := ts.nfs.NFSPROC3_WRITE(args)
	return &reply
}

func (ts *TestState) WriteOff(fh nfstypes.Nfs_fh3, off uint64, data []byte, how nfstypes.Stable_how) {
	reply := ts.WriteOp(fh, off, data, how)
	assert.Equal(ts.t, reply.Status, nfstypes.NFS3_OK)
	assert.Equal(ts.t, reply.Resok.Count, nfstypes.Count3(len(data)))
}

func (ts *TestState) WriteErr(fh nfstypes.Nfs_fh3, data []byte, how nfstypes.Stable_how, err nfstypes.Nfsstat3) {
	reply := ts.WriteOp(fh, 0, data, how)
	assert.Equal(ts.t, reply.Status, err)
}

func (ts *TestState) Write(fh nfstypes.Nfs_fh3, data []byte, how nfstypes.Stable_how) {
	ts.WriteOff(fh, uint64(0), data, how)
}

func (ts *TestState) Read(fh nfstypes.Nfs_fh3, off uint64, sz uint64) []byte {
	args := nfstypes.READ3args{
		File:   fh,
		Offset: nfstypes.Offset3(off),
		Count:  nfstypes.Count3(sz)}
	reply := ts.nfs.NFSPROC3_READ(args)
	assert.Equal(ts.t, reply.Status, nfstypes.NFS3_OK)
	assert.Equal(ts.t, reply.Resok.Count, nfstypes.Count3(sz))
	return reply.Resok.Data
}

func (ts *TestState) Remove(name string) {
	what := nfstypes.Diropargs3{Dir: fh.MkRootFh3(), Name: nfstypes.Filename3(name)}
	args := nfstypes.REMOVE3args{
		Object: what,
	}
	reply := ts.nfs.NFSPROC3_REMOVE(args)
	assert.Equal(ts.t, nfstypes.NFS3_OK, reply.Status)
}

func (ts *TestState) MkDir(name string) {
	where := nfstypes.Diropargs3{Dir: fh.MkRootFh3(), Name: nfstypes.Filename3(name)}
	sattr := nfstypes.Sattr3{}
	args := nfstypes.MKDIR3args{Where: where, Attributes: sattr}
	attr := ts.nfs.NFSPROC3_MKDIR(args)
	assert.Equal(ts.t, nfstypes.NFS3_OK, attr.Status)
}

func (ts *TestState) ReadDirPlus() nfstypes.Dirlistplus3 {
	args := nfstypes.READDIRPLUS3args{Dir: fh.MkRootFh3(), Dircount: nfstypes.Count3(100), Maxcount: nfstypes.Count3(inode.NDIRECT * disk.BlockSize)}
	reply := ts.nfs.NFSPROC3_READDIRPLUS(args)
	assert.Equal(ts.t, reply.Status, nfstypes.NFS3_OK)
	return reply.Resok.Reply
}

func (ts *TestState) CommitOp(fh nfstypes.Nfs_fh3, cnt uint64) *nfstypes.COMMIT3res {
	args := nfstypes.COMMIT3args{
		File:   fh,
		Offset: nfstypes.Offset3(0),
		Count:  nfstypes.Count3(cnt)}
	reply := ts.nfs.NFSPROC3_COMMIT(args)
	return &reply
}

func (ts *TestState) Commit(fh nfstypes.Nfs_fh3, cnt uint64) {
	reply := ts.CommitOp(fh, cnt)
	assert.Equal(ts.t, reply.Status, nfstypes.NFS3_OK)
}

func (ts *TestState) CommitErr(fh nfstypes.Nfs_fh3, cnt uint64, err nfstypes.Nfsstat3) {
	reply := ts.CommitOp(fh, cnt)
	assert.Equal(ts.t, reply.Status, err)
}

func (ts *TestState) RenameOp(fhfrom nfstypes.Nfs_fh3, from string,
	fhto nfstypes.Nfs_fh3, to string) nfstypes.Nfsstat3 {
	args := nfstypes.RENAME3args{
		From: nfstypes.Diropargs3{Dir: fhfrom, Name: nfstypes.Filename3(from)},
		To:   nfstypes.Diropargs3{Dir: fhto, Name: nfstypes.Filename3(to)},
	}
	reply := ts.nfs.NFSPROC3_RENAME(args)
	return reply.Status
}

func (ts *TestState) Rename(from string, to string) {
	status := ts.RenameOp(fh.MkRootFh3(), from, fh.MkRootFh3(), to)
	assert.Equal(ts.t, status, nfstypes.NFS3_OK)
}

func (ts *TestState) RenameFhs(fhfrom nfstypes.Nfs_fh3, from string, fhto nfstypes.Nfs_fh3, to string) {
	status := ts.RenameOp(fhfrom, from, fhto, to)
	assert.Equal(ts.t, status, nfstypes.NFS3_OK)
}

func (ts *TestState) RenameFail(from string, to string) {
	status := ts.RenameOp(fh.MkRootFh3(), from, fh.MkRootFh3(), to)
	assert.Equal(ts.t, status, nfstypes.NFS3ERR_NOTEMPTY)
}

func mkdata(sz uint64) []byte {
	data := make([]byte, sz)
	for i := range data {
		data[i] = byte(i % 128)
	}
	return data
}

func mkdataval(b byte, sz uint64) []byte {
	data := make([]byte, sz)
	for i := range data {
		data[i] = b
	}
	return data
}

func (ts *TestState) readcheck(fh nfstypes.Nfs_fh3, off uint64, data []byte) {
	d := ts.Read(fh, off, uint64(len(data)))
	assert.Equal(ts.t, len(data), len(d))
	for i := uint64(0); i < uint64(len(data)); i++ {
		assert.Equal(ts.t, data[i], d[i])
	}
}

func newTest(t *testing.T) *TestState {
	checkFlags()
	fmt.Printf("%s\n", t.Name())
	return &TestState{t: t, nfs: MkNfs()}
}

func (ts *TestState) Close() {
	ts.nfs.ShutdownNfs()
	fmt.Printf("%s\n", ts.t.Name())
}

func TestRoot(t *testing.T) {
	fmt.Printf("TestGetRoot\n")
	ts := newTest(t)
	defer ts.Close()

	fh := fh.MkRootFh3()
	ts.GetattrDir(fh)
	fhdot := ts.LookupFh(fh, ".")
	ts.GetattrDir(fhdot)
	fhdotdot := ts.LookupFh(fh, "..")
	ts.GetattrDir(fhdotdot)
	// assert.Equal(ts.t, fhdot.ino, fhdotdot.ino)
}

func TestReadDir(t *testing.T) {
	checkFlags()
	ts := newTest(t)

	dl3 := ts.ReadDirPlus()
	ne3 := dl3.Entries
	for ne3 != nil {
		assert.Equal(t, ne3.Fileid, nfstypes.Fileid3(1))
		ne3 = ne3.Nextentry
	}
}

// Grow file with setattr before writing
func TestOneFile(t *testing.T) {
	ts := newTest(t)
	sz := uint64(8192)
	ts.Create("x")
	fh := ts.Lookup("x", true)
	ts.Getattr(fh, 0)
	data := mkdata(sz)
	ts.Setattr(fh, sz)
	ts.Getattr(fh, sz)
	ts.Write(fh, data, nfstypes.FILE_SYNC)
	ts.readcheck(fh, 0, data)
	ts.Remove("x")
	_ = ts.Lookup("x", false)
	ts.GetattrFail(fh)
}

// Grow file by writing
func TestFile1(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()

	sz := uint64(122)
	ts.Create("x")
	fh := ts.Lookup("x", true)
	data := mkdata(uint64(sz))
	ts.Write(fh, data, nfstypes.FILE_SYNC)
	ts.readcheck(fh, 0, data)
}

func TestOneDir(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()

	ts.MkDir("d")
	fh := ts.Lookup("d", true)
	ts.GetattrDir(fh)

	// . and ..
	fhdot := ts.LookupFh(fh, ".")
	ts.GetattrDir(fhdot)
	fhdotdot := ts.LookupFh(fh, "..")
	ts.GetattrDir(fhdotdot)

	ts.Rename("d", "d1")
	_ = ts.Lookup("d", false)
	fh = ts.Lookup("d1", true)
	ts.GetattrDir(fh)

	// rename d1 into an existing, empty dir d2
	ts.MkDir("d2")
	fh = ts.Lookup("d2", true)
	ts.GetattrDir(fh)
	ts.Rename("d1", "d2")
	_ = ts.Lookup("d1", false)
	fh = ts.Lookup("d2", true)
	ts.GetattrDir(fh)

	// rename into non-empty dir d3
	ts.MkDir("d3")
	fh3 := ts.Lookup("d3", true)
	ts.GetattrDir(fh3)
	ts.CreateFh(fh3, "f")
	ts.RenameFail("d2", "d3")

	// Rmdir("d")
}

// Many filesgg
func TestManyFiles(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()

	for i := 0; i < 100; i++ {
		ts.Create("x")
		ts.Remove("x")
	}
}

// Create many files and then delete
func TestManyFiles1(t *testing.T) {
	const N = 50
	ts := newTest(t)
	defer ts.Close()

	for i := 0; i < N; i++ {
		s := strconv.Itoa(i)
		ts.Create("x" + s)
	}
	for i := 0; i < N; i++ {
		s := strconv.Itoa(i)
		ts.Remove("x" + s)
	}
}

func TestRename(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()
	ts.Create("x")
	ts.Rename("x", "y")
	_ = ts.Lookup("x", false)
	fh := ts.Lookup("y", true)
	ts.Getattr(fh, 0)

	ts.MkDir("d1")
	d1 := ts.Lookup("d1", true)
	ts.GetattrDir(d1)
	ts.CreateFh(d1, "f1")
	ts.LookupFh(d1, "f1")

	ts.MkDir("d2")
	d2 := ts.Lookup("d2", true)
	ts.GetattrDir(d2)

	ts.RenameFhs(d1, "f1", d2, "f1")
}

func TestUnstable(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()
	ts.Create("x")
	ts.Create("y")
	sz := uint64(4096)
	x := ts.Lookup("x", true)
	y := ts.Lookup("y", true)

	// This write will stay in memory log
	data1 := mkdataval(1, sz)
	ts.Write(x, data1, nfstypes.UNSTABLE)

	ts.Write(y, data1, nfstypes.FILE_SYNC)

	data2 := mkdataval(2, sz)
	ts.Write(x, data2, nfstypes.UNSTABLE)
	ts.Commit(x, sz)

	ts.readcheck(x, 0, data2)
}

func TestConcurWrite(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()

	names := []string{"f0", "f1", "f3", "f4"}
	const N uint64 = 32
	const SZ = disk.BlockSize
	var wg sync.WaitGroup
	for g, n := range names {
		ts.Create(n)
		fh := ts.Lookup(n, true)
		wg.Add(1)
		go func(fh nfstypes.Nfs_fh3, v byte) {
			for i := uint64(0); i < N; i++ {
				data := mkdataval(v, SZ)
				ts.WriteOff(fh, i*SZ, data, nfstypes.FILE_SYNC)
			}
			wg.Done()
		}(fh, byte(g))
	}

	wg.Wait()
	for g, n := range names {
		fh := ts.Lookup(n, true)
		buf := ts.Read(fh, 0, N*SZ)
		assert.Equal(t, N*SZ, uint64(len(buf)))
		for _, v := range buf {
			assert.Equal(t, byte(g), v)
		}
	}
}

func TestConcurCreateDelete(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()

	names := []string{"f0", "f1", "f3", "f4"}
	const N = 10
	var wg sync.WaitGroup
	for _, n := range names {
		wg.Add(1)
		go func(n string) {
			for i := 0; i < N; i++ {
				s := strconv.Itoa(i)
				ts.Create(n + s)
				if i > 0 && (i%2) == 0 {
					s := strconv.Itoa(i / 2)
					ts.Remove(n + s)
				}
			}
			wg.Done()
		}(n)
	}
	wg.Wait()
	for _, n := range names {
		for i := 0; i < N; i++ {
			s := strconv.Itoa(i)
			if i > 0 && i < N/2 {
				ts.Lookup(n+s, false)
			} else {
				ts.Lookup(n+s, true)
			}
		}
	}
}

func TestConcurRename(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()

	const NGO = 4
	const N = 20
	var wg sync.WaitGroup

	for i := 0; i < NGO; i++ {
		wg.Add(1)
		go func(id int) {
			for i := 0; i < N; i++ {
				from := "f" + strconv.Itoa(id)
				to := "g" + strconv.Itoa(id)
				ts.Create(from)
				ts.Rename(from, to)
				ts.Remove(to)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func TestFileHole(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()

	sz := uint64(122)
	ts.Create("x")
	fh := ts.Lookup("x", true)

	data := mkdata(uint64(sz))
	ts.WriteOff(fh, 4096, data, nfstypes.FILE_SYNC)

	null := mkdataval(0, 4096)
	ts.readcheck(fh, 0, null)
}

func (ts *TestState) evict(names []string) {
	const N uint64 = BCACHESZ * uint64(10)
	var wg sync.WaitGroup
	for _, n := range names {
		wg.Add(1)
		go func(n string) {
			ts.Create(n)
			sz := uint64(4096)
			x := ts.Lookup(n, true)
			for i := uint64(0); i < N; i++ {
				data1 := mkdataval(1, sz)
				ts.WriteOff(x, i*sz, data1, nfstypes.UNSTABLE)
			}
			ts.Commit(x, sz*N)
			wg.Done()
		}(n)
	}
	wg.Wait()
}

func TestSerialEvict(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()

	ts.evict([]string{"f0"})
}

func TestConcurEvict(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()
	const N = 10

	names := make([]string, N)
	for i := 0; i < 10; i++ {
		names[i] = "f" + strconv.Itoa(i)
	}

	ts.evict(names)
}

func TestLargeFile(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()
	const N = 522

	ts.Create("x")
	sz := uint64(4096)
	x := ts.Lookup("x", true)
	for i := uint64(0); i < N; i++ {
		data := mkdataval(byte(i), sz)
		ts.WriteOff(x, i*sz, data, nfstypes.UNSTABLE)
	}
	ts.Commit(x, sz*N)

	for i := uint64(0); i < N; i++ {
		data := mkdataval(byte(i), sz)
		ts.readcheck(x, i*sz, data)
	}
	ts.Remove("x")
}

func TestBigWrite(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()

	ts.Create("x")
	sz := uint64(4096 * (fs.HDRADDRS / 2))
	x := ts.Lookup("x", true)
	data := mkdataval(byte(0), sz)
	ts.Write(x, data, nfstypes.UNSTABLE)
	ts.Commit(x, sz)

	// Too big
	ts.Create("y")
	sz = uint64(4096 * (fs.HDRADDRS + 10))
	y := ts.Lookup("y", true)
	data = mkdataval(byte(0), sz)
	ts.WriteErr(y, data, nfstypes.UNSTABLE, nfstypes.NFS3ERR_INVAL)
	ts.CommitErr(y, sz, nfstypes.NFS3ERR_INVAL)
}

func TestBigUnlink(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()
	const N = 100 * (disk.BlockSize / 8)

	ts.Create("x")
	sz := disk.BlockSize
	x := ts.Lookup("x", true)
	for i := uint64(0); i < N; i++ {
		data := mkdataval(byte(i), sz)
		ts.WriteOff(x, i*sz, data, nfstypes.UNSTABLE)
	}
	ts.Commit(x, sz*N)

	ts.Remove("x")
}

func BenchmarkSmall(b *testing.B) {
	data := mkdata(uint64(100))
	ts := &TestState{t: nil, nfs: MkNfs()}
	for i := 0; i < b.N; i++ {
		s := strconv.Itoa(i)
		name := "x" + s
		ts.CreateOp(fh.MkRootFh3(), "x"+s)
		reply := ts.LookupOp(fh.MkRootFh3(), name)
		if reply.Status != nfstypes.NFS3_OK {
			panic("BenchmarkSmall")
		}
		ts.WriteOp(reply.Resok.Object, 0, data, nfstypes.FILE_SYNC)
	}
}
