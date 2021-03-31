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
./loc.py
```

## Gather data

```sh
bench.sh | tee bench-raw.txt`
```

takes about a minute

```sh
./scale.sh 10 | tee scale-raw.txt
```

takes a few minutes

## Produce graphs

3. Produce graphs

```sh
./bench.py bench-raw.txt
./scale.py scale-raw.txt
```

TODO: copy in gnuplot scripts
