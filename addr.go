package goose_nfs

type addr struct {
	blkno uint64
	off   uint64 // offset in bits
	sz    uint64 // sz in bits
}

func (a *addr) eq(b addr) bool {
	return a.blkno == b.blkno && a.off == b.off && a.sz == b.sz
}

func mkaddr(blkno uint64, off uint64, sz uint64) addr {
	return addr{blkno: blkno, off: off, sz: sz}
}

//
// a map from addr to an object
//
type aentry struct {
	addr addr
	obj  interface{}
}

type addrMap struct {
	addrs map[uint64][]*aentry
}

func mkAddrMap() *addrMap {
	a := &addrMap{
		addrs: make(map[uint64][]*aentry),
	}
	return a
}

func (amap *addrMap) lookup(addr addr) interface{} {
	var obj interface{}
	addrs, ok := amap.addrs[addr.blkno]
	if ok {
		for _, a := range addrs {
			if addr.eq(a.addr) {
				obj = a.obj
				break
			}
		}
	}
	return obj
}

func (amap *addrMap) insert(addr addr, obj interface{}) {
	aentry := &aentry{addr: addr, obj: obj}
	blkno := addr.blkno
	amap.addrs[blkno] = append(amap.addrs[blkno], aentry)
}

func (amap *addrMap) del(addr addr) {
	var index uint64
	var found bool

	blkno := addr.blkno
	locks, found := amap.addrs[blkno]
	if !found {
		panic("release")
	}
	for i, l := range locks {
		if l.addr.eq(addr) {
			index = uint64(i)
			found = true
		}
	}
	if !found {
		panic("release")
	}
	locks = append(locks[0:index], locks[index+1:]...)
	amap.addrs[blkno] = locks
}

func (amap *addrMap) apply(f func(addr, interface{})) {
	for _, addrs := range amap.addrs {
		for _, a := range addrs {
			f(a.addr, a.obj)
		}
	}
}
