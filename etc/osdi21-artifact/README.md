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

```sh
./plot.sh
```

(assumes you've put data in `data/bench-raw.txt` and `data/scale-raw.txt`, as
above)
