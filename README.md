# goose-nfsd

[![Build Status](https://travis-ci.com/mit-pdos/goose-nfsd.svg?token=1SPwqpqUkmsUej6KT47u&branch=master)](https://travis-ci.com/mit-pdos/goose-nfsd)

An NFS implementation that works with [Goose](https://github.com/tchajed/goose).

## Remaining conversion issues

```
conversion failed: [unsupported]: non-whitelisted import
"fmt"
  /Users/tchajed/code/goose/goose.go:1868
  src: buf.go:6:2

[unsupported]: cannot call methods selected from fmt
fmt.Sprintf
  /Users/tchajed/code/goose/goose.go:555
  src: buf.go:45:9

[unsupported]: non-whitelisted import
"fmt"
  /Users/tchajed/code/goose/goose.go:1868
  src: inode.go:7:2

[unsupported]: cannot call methods selected from fmt
fmt.Sprintf
  /Users/tchajed/code/goose/goose.go:555
  src: inode.go:62:9

[future]: return in loop (use break)
return data, false
  /Users/tchajed/code/goose/goose.go:1707
  src: inode.go:349:4

[unsupported]: non-whitelisted import
"sort"
  /Users/tchajed/code/goose/goose.go:1868
  src: nfs_ops.go:4:2

[unsupported]: cannot call methods selected from sort
sort.Slice
  /Users/tchajed/code/goose/goose.go:555
  src: nfs_ops.go:142:2

[future]: return in loop (use break)
return 0, false
  /Users/tchajed/code/goose/goose.go:1707
  src: txn.go:185:4

[unsupported]: unexpected type expr
...interface{}
  /Users/tchajed/code/goose/goose.go:293
  src: util.go:7:45

9 errors
```
