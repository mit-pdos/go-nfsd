# GoNFS

[![CI](https://github.com/mit-pdos/go-nfsd/actions/workflows/build.yml/badge.svg)](https://github.com/mit-pdos/go-nfsd/actions/workflows/build.yml)

An NFSv3 server that uses [GoJournal](https://github.com/mit-pdos/go-journal) to
write operations atomically to disk. Unlike many other NFS servers, GoNFS
implements the file-system abstraction on top of a disk (which can be any file,
including an in-memory file in `/tmp`, an ordinary file in another file system,
or a block device).

## GoJournal artifact

The artifact for the OSDI 2021 GoJournal paper is in this repo at
[artifact](artifact/). See the README there for detailed instructions on
obtaining and using the artifact.
