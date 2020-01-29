package goose_nfs

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"strconv"
	"sync"

	"github.com/tchajed/goose/machine/disk"

	"testing"

	"github.com/mit-pdos/goose-nfsd/bcache"
	"github.com/mit-pdos/goose-nfsd/dir"
	"github.com/mit-pdos/goose-nfsd/fh"
	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/nfstypes"

	"github.com/stretchr/testify/assert"
)

var quiet = flag.Bool("quiet", false, "disable logging")

const DISKSZ uint64 = 10 * 1000

func checkFlags() {
	if *quiet {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}
}

type TestState struct {
	t    *testing.T
	clnt *NfsClient
}

func (ts *TestState) CreateFh(fh nfstypes.Nfs_fh3, name string) {
	attr := ts.clnt.CreateOp(fh, name)
	assert.Equal(ts.t, nfstypes.NFS3_OK, attr.Status)
}

func (ts *TestState) Create(name string) {
	ts.CreateFh(fh.MkRootFh3(), name)
}

func (ts *TestState) LookupFh(fh nfstypes.Nfs_fh3, name string) nfstypes.Nfs_fh3 {
	reply := ts.clnt.LookupOp(fh, name)
	assert.Equal(ts.t, reply.Status, nfstypes.NFS3_OK)
	return reply.Resok.Object
}

func (ts *TestState) Lookup(name string, succeed bool) nfstypes.Nfs_fh3 {
	reply := ts.clnt.LookupOp(fh.MkRootFh3(), name)
	if succeed {
		assert.Equal(ts.t, reply.Status, nfstypes.NFS3_OK)
	} else {
		assert.NotEqual(ts.t, reply.Status, nfstypes.NFS3_OK)
	}
	return reply.Resok.Object
}

func (ts *TestState) Getattr(fh nfstypes.Nfs_fh3, sz uint64) {
	attr := ts.clnt.GetattrOp(fh)
	assert.Equal(ts.t, nfstypes.NFS3_OK, attr.Status)
	assert.Equal(ts.t, nfstypes.NF3REG, attr.Resok.Obj_attributes.Ftype)
	assert.Equal(ts.t, nfstypes.Size3(sz), attr.Resok.Obj_attributes.Size)
}

func (ts *TestState) GetattrDir(fh nfstypes.Nfs_fh3) nfstypes.Fattr3 {
	attr := ts.clnt.GetattrOp(fh)
	assert.Equal(ts.t, nfstypes.NFS3_OK, attr.Status)
	assert.Equal(ts.t, attr.Resok.Obj_attributes.Ftype, nfstypes.NF3DIR)
	return attr.Resok.Obj_attributes
}

func (ts *TestState) GetattrFail(fh nfstypes.Nfs_fh3) {
	attr := ts.clnt.GetattrOp(fh)
	assert.Equal(ts.t, attr.Status, nfstypes.NFS3ERR_STALE)
}

func (ts *TestState) Setattr(fh nfstypes.Nfs_fh3, sz uint64) {
	reply := ts.clnt.SetattrOp(fh, sz)
	assert.Equal(ts.t, reply.Status, nfstypes.NFS3_OK)
}

func (ts *TestState) WriteOff(fh nfstypes.Nfs_fh3, off uint64, data []byte, how nfstypes.Stable_how) {
	reply := ts.clnt.WriteOp(fh, off, data, how)
	assert.Equal(ts.t, nfstypes.NFS3_OK, reply.Status)
	assert.Equal(ts.t, nfstypes.Count3(len(data)), reply.Resok.Count)
}

func (ts *TestState) WriteErr(fh nfstypes.Nfs_fh3, data []byte, how nfstypes.Stable_how, err nfstypes.Nfsstat3) {
	reply := ts.clnt.WriteOp(fh, 0, data, how)
	assert.Equal(ts.t, err, reply.Status)
}

func (ts *TestState) Write(fh nfstypes.Nfs_fh3, data []byte, how nfstypes.Stable_how) {
	ts.WriteOff(fh, uint64(0), data, how)
}

func (ts *TestState) Read(fh nfstypes.Nfs_fh3, off uint64, sz uint64) []byte {
	reply := ts.clnt.ReadOp(fh, off, sz)
	assert.Equal(ts.t, nfstypes.NFS3_OK, reply.Status)
	assert.Equal(ts.t, nfstypes.Count3(sz), reply.Resok.Count)
	return reply.Resok.Data
}

func (ts *TestState) Remove(name string) {
	reply := ts.clnt.RemoveOp(fh.MkRootFh3(), name)
	assert.Equal(ts.t, nfstypes.NFS3_OK, reply.Status)
}

func (ts *TestState) MkDir(name string) {
	attr := ts.clnt.MkDirOp(fh.MkRootFh3(), name)
	assert.Equal(ts.t, nfstypes.NFS3_OK, attr.Status)
}

func (ts *TestState) RmDir(name string, err nfstypes.Nfsstat3) {
	attr := ts.clnt.RmDirOp(fh.MkRootFh3(), name)
	assert.Equal(ts.t, err, attr.Status)
}

func (ts *TestState) SymLink(name string, target string) {
	attr := ts.clnt.SymLinkOp(fh.MkRootFh3(), name, nfstypes.Nfspath3(target))
	assert.Equal(ts.t, nfstypes.NFS3_OK, attr.Status)
}

func (ts *TestState) ReadLink(fh nfstypes.Nfs_fh3) string {
	attr := ts.clnt.ReadLinkOp(fh)
	assert.Equal(ts.t, nfstypes.NFS3_OK, attr.Status)
	return string(attr.Resok.Data)
}

func (ts *TestState) ReadDirPlus() nfstypes.Dirlistplus3 {
	reply := ts.clnt.ReadDirPlusOp(fh.MkRootFh3(), inode.NDIRECT*disk.BlockSize)
	assert.Equal(ts.t, reply.Status, nfstypes.NFS3_OK)
	return reply.Resok.Reply
}

func (ts *TestState) Commit(fh nfstypes.Nfs_fh3, cnt uint64) {
	reply := ts.clnt.CommitOp(fh, cnt)
	assert.Equal(ts.t, reply.Status, nfstypes.NFS3_OK)
}

func (ts *TestState) CommitErr(fh nfstypes.Nfs_fh3, cnt uint64, err nfstypes.Nfsstat3) {
	reply := ts.clnt.CommitOp(fh, cnt)
	assert.Equal(ts.t, reply.Status, err)
}

func (ts *TestState) Rename(from string, to string) {
	status := ts.clnt.RenameOp(fh.MkRootFh3(), from, fh.MkRootFh3(), to)
	assert.Equal(ts.t, status, nfstypes.NFS3_OK)
}

func (ts *TestState) RenameFhs(fhfrom nfstypes.Nfs_fh3, from string, fhto nfstypes.Nfs_fh3, to string) {
	status := ts.clnt.RenameOp(fhfrom, from, fhto, to)
	assert.Equal(ts.t, status, nfstypes.NFS3_OK)
}

func (ts *TestState) RenameFail(from string, to string) {
	status := ts.clnt.RenameOp(fh.MkRootFh3(), from, fh.MkRootFh3(), to)
	assert.Equal(ts.t, nfstypes.NFS3ERR_NOTEMPTY, status)
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
	return &TestState{t: t, clnt: MkNfsClient(DISKSZ)}
}

func (ts *TestState) Close() {
	ts.clnt.ShutdownDestroy()
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
}

func TestReadDir(t *testing.T) {
	checkFlags()
	ts := newTest(t)
	defer ts.Close()

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
	defer ts.Close()

	sz := uint64(8192)
	ts.Create("x")
	attr := ts.GetattrDir(fh.MkRootFh3())
	assert.Equal(t, 3*dir.DIRENTSZ, uint64(attr.Size))
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
	ts.Getattr(fh, sz)
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
	ts.LookupFh(fh3, "f")
	ts.RenameFail("d2", "d3")

	ts.RmDir("d", nfstypes.NFS3ERR_NOENT)
	ts.RmDir("d2", nfstypes.NFS3_OK)
	ts.RmDir("d3", nfstypes.NFS3ERR_INVAL)
}

// Many files
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

func TestSymLink(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()

	ts.Create("x")
	ts.SymLink("y", "x")
	fh := ts.Lookup("y", true)
	p := ts.ReadLink(fh)
	assert.Equal(ts.t, "x", p)
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

func TestConcurWriteFiles(t *testing.T) {
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

func TestConcurWriteBlocks(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()
	const SZ = 100
	const N = 4

	n := "x"
	ts.Create(n)
	fh := ts.Lookup(n, true)
	var wg sync.WaitGroup
	for t := 0; t < N; t++ {
		wg.Add(1)
		go func(fh nfstypes.Nfs_fh3, t uint64) {
			for i := t; i < SZ*N; i += N {
				data := mkdataval(byte(t), disk.BlockSize)
				ts.WriteOff(fh, uint64(i)*disk.BlockSize, data,
					nfstypes.UNSTABLE)
			}
			wg.Done()
		}(fh, uint64(t))
	}
	ts.Commit(fh, 0)
	wg.Wait()
	for i := 0; i < SZ*N; i++ {
		fh := ts.Lookup(n, true)
		buf := ts.Read(fh, uint64(i)*disk.BlockSize, disk.BlockSize)
		for _, v := range buf {
			assert.Equal(t, v, byte(i%N))
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

func TestManyHoles(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()

	sz := uint64(8192)
	data := mkdata(uint64(sz))
	n := 4
	for i := 0; i < 50; i++ {
		ts.Create("x")
		fh := ts.Lookup("x", true)
		for j := 0; j < n; j++ {
			off := rand.Uint64()
			off = off % (inode.MaxFileSize() - sz)
			ts.WriteOff(fh, off, data, nfstypes.FILE_SYNC)
		}
		ts.Remove("x")
	}
}

func (ts *TestState) evict(names []string) {
	const N uint64 = bcache.BCACHESZ * 2
	var wg sync.WaitGroup
	if N*uint64(len(names)) > DISKSZ {
		panic("Disk is too small")
	}
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
	const N = 4

	names := make([]string, N)
	for i := 0; i < N; i++ {
		names[i] = "f" + strconv.Itoa(i)
	}

	ts.evict(names)
}

func TestWriteLargeFile(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()
	// allocate a double indirect block
	const N = inode.NDIRECT + disk.BlockSize/8 + 10

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
	const N = DISKSZ / 2

	for j := 0; j < 4; j++ {
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
}

func (ts *TestState) maketoolargefile(name string, wsize int) {
	ts.Create(name)
	sz := uint64(4096 * wsize)
	x := ts.Lookup(name, true)
	for i := uint64(0); ; {
		data := mkdataval(byte(i%128), sz)
		reply := ts.clnt.WriteOp(x, i, data, nfstypes.FILE_SYNC)
		if reply.Status == nfstypes.NFS3_OK {
			assert.LessOrEqual(ts.t, uint64(reply.Resok.Count), uint64(len(data)))
		} else {
			assert.Equal(ts.t, reply.Status, nfstypes.NFS3ERR_NOSPC)
			break
		}
		i += uint64(reply.Resok.Count)
	}
}

func TestTooLargeFile(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()

	ts.maketoolargefile("x", 50)
	ts.Remove("x")
}

func TestRestart(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()

	ts.Create("x")
	ts.clnt.Shutdown()
	ts.clnt.srv = MakeNfs(ts.clnt.srv.Name, DISKSZ)
	ts.Lookup("x", true)
	ts.Create("y")
	ts.Lookup("y", true)
}

func TestAbort(t *testing.T) {
	ts := newTest(t)
	defer ts.Close()

	ts.maketoolargefile("x", 50)
	// an inode for d can allocated but there is no block for the dir
	// mkdir will allocate inode 3
	attr := ts.clnt.MkDirOp(fh.MkRootFh3(), "d")
	assert.Equal(ts.t, nfstypes.NFS3ERR_NOSPC, attr.Status)
	// d better not exist
	ts.Lookup("d", false)
	ts.clnt.Shutdown()
	ts.clnt.srv = MakeNfs(ts.clnt.srv.Name, DISKSZ)
	ts.Remove("x")
	ts.MkDir("d1") // reallocate inode 2 (x's inode)
	ts.MkDir("d")
	fh3 := ts.Lookup("d", true)
	fh := fh.MakeFh(fh3)
	// inode 3 should be used for d
	assert.Equal(ts.t, fh.Ino, fs.Inum(3))
}
