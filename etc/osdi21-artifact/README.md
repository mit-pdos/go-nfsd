# JrnlCert artifact

## Set environment variables

```sh
export PERENNIAL_PATH=$HOME/code/perennial
export GOOSE_NFSD_PATH=$HOME/code/goose-nfsd
export MARSHAL_PATH=$HOME/code/marshal
export XV6_PATH=$HOME/code/xv6-public
```

## Lines of code table

```sh
./loc.py | tee data/lines-of-code.txt
```

## Gather data

```sh
bench.sh | tee data/bench-raw.txt`
```

takes about a minute

```sh
./scale.sh 10 | tee data/scale-raw.txt
```

takes a few minutes

## Produce graphs

3. Produce graphs

```sh
./bench.py data/bench-raw.txt
./scale.py data/scale-raw.txt
gnuplot bench.plot
gnuplot scale.plot
```

TODO: run serial NFS workload
