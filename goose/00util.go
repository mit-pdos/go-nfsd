package wal

const Debug uint64 = 0

func RoundUp(n uint64, sz uint64) uint64 {
	return (n + sz - 1) / sz
}

func Min(n uint64, m uint64) uint64 {
	if n < m {
		return n
	} else {
		return m
	}
}

func xxcopy(dst []byte, src []byte) {
	dlen := uint64(len(dst))
	slen := uint64(len(src))
	copylen := Min(dlen, slen)
	for i := uint64(0); i < copylen; i++ {
		dst[i] = src[i]
	}
}
