# JrnlCert artifact

## About the VM

The VM was created by installing the Ubuntu 20.04 live server image. The user
account is `ubuntu` with password `ubuntu`, and the hostname is `jrnlcert-vm`.
The user account has sudo access without a password.

You can launch the VM headless and then SSH to it. There's a port forwarding
rule set up in VirtualBox so that `ssh -p 10322 ubuntu@localhost` should work.
You might want to add your public key to the VM to avoid having to type the
password every time.

## Compiling the proofs

The paper claims to have verified the GoJrnl implementation. You should check
this by compiling the proofs in the `perennial` repo.

## Performance evaluation

### Set environment variables

```sh
export PERENNIAL_PATH=$HOME/code/perennial
export GOOSE_NFSD_PATH=$HOME/code/goose-nfsd
export MARSHAL_PATH=$HOME/code/marshal
export XV6_PATH=$HOME/code/xv6-public
```

### Gather data

```sh
./loc.py | tee data/lines-of-code.txt
```

instantaneous

```sh
bench.sh | tee data/bench-raw.txt`
```

takes about a minute

```sh
./scale.sh 10 | tee data/scale-raw.txt
```

takes a few minutes

### Produce graphs

```sh
./plot.sh
```

(assumes you've put data in `data/bench-raw.txt` and `data/scale-raw.txt`, as
above)

### Check the output

Compare `data/lines-of-code.txt` to Figures 14 and 15 in the paper. The exact
lines won't be the same because the code has changed slightly (and this artifact
is automated differently from how the original data was gathered), but the
numbers should generally line up.

The exact performance results will vary depending on your machine, and suffer
slightly from being run in a VM.

Compare `fig/bench.png` to Figure 16 in the paper. The absolute performance
numbers were included manually in the graph; you can easily find the numbers by
looking at `data/bench.data` and looking at the "linux" column.

Compare `fig/scale.png` to Figure 17 in the paper. The scaling should be roughly
the same. Absolute performance depends highly on your drive's performance.
