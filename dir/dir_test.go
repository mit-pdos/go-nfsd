package dir

import (
	"testing"

	"github.com/mit-pdos/goose-nfsd/nfstypes"
	"github.com/zeldovich/go-rpcgen/xdr"
)

func TestFattr3Size(t *testing.T) {
	var e nfstypes.Fattr3
	bs, err := xdr.EncodeBuf(&e)
	if err != nil {
		panic(err)
	}
	if len(bs) != int(fattr3XDRsize) {
		t.Fatalf("size of fattr3 is %d != %d", len(bs), fattr3XDRsize)
	}
}

func TestNameBaggage(t *testing.T) {
	var e nfstypes.Filename3
	bs, err := xdr.EncodeBuf(&e)
	if err != nil {
		panic(err)
	}
	if len(bs) != 4 {
		t.Fatalf("size of empty filename is %d != 4", len(bs))
	}
}

func TestEntryPlus3Baggage(t *testing.T) {
	var e nfstypes.Entryplus3
	e.Name_attributes.Attributes_follow = true
	e.Name_handle.Handle_follows = true
	bs, err := xdr.EncodeBuf(&e)
	if err != nil {
		panic(err)
	}
	if len(bs) > int(entryplus3Baggage) {
		t.Fatalf("size of entryplus3 is %d > %d", len(bs), entryplus3Baggage)
	}
}
