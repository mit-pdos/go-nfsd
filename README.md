# GoNFS

[![CI](https://github.com/mit-pdos/go-nfsd/actions/workflows/build.yml/badge.svg)](https://github.com/mit-pdos/go-nfsd/actions/workflows/build.yml)

An NFSv3 server that uses [GoJournal](https://github.com/mit-pdos/go-journal) to
write operations atomically to disk. Unlike many other NFS servers, GoNFS
implements the file-system abstraction on top of a disk (which can be any file,
including an in-memory file in `/tmp`, an ordinary file in another file system,
or a block device).
