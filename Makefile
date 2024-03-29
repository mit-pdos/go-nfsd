GOPATH		:= $(shell go env GOPATH)
GOOSE_DIRS	:= super cache fh fstxn kvs nfstypes simple

# Things that don't goose yet:
#   .
#   dcache: map with string keys
#   inode: time package
#   nfstypes: need to ignore nfs_xdr.go
#   dir

COQ_PKGDIR := Goose/github_com/mit_pdos/go_nfsd

all: check goose-output

check:
	test -z "$$(gofmt -d .)"
	go vet ./...

goose-output: $(patsubst %,${COQ_PKGDIR}/%.v,$(GOOSE_DIRS))

${COQ_PKGDIR}/%.v: % %/*
	$(GOPATH)/bin/goose -out Goose ./$<

clean:
	rm -rf Goose
