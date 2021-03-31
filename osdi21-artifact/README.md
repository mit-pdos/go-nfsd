# JrnlCert artifact

## About the VM

The VM was created by installing the Ubuntu 20.04 live server image. The user
account is `ubuntu` with password `ubuntu`, and the hostname is `jrnlcert-vm`.
The user account has sudo access without a password.

You can launch the VM headless and then SSH to it. There's a port forwarding
rule set up in VirtualBox so that `ssh -p 10322 ubuntu@localhost` should work.
You might want to add your public key to the VM to avoid having to type the
password every time.

Note that GoNFS from the paper is called goose-nfsd in the code, and that
JrnlCert is an anonymized name for the new version of the Perennial framework.

We've configured the VM with 4GB of RAM, but if you can afford 6GB or 8GB you
may want to do that to speed up compilation (note that single-threaded
compilation will take just over an hour and doesn't require much RAM, so more
RAM isn't necessary). You'll definitely want to set it to the maximum number of
cores you can afford to run the scalability experiment.

## Compiling the proofs

The paper claims to have verified the GoJrnl implementation. You should check
this by compiling the proofs in the `perennial` repo:

```sh
cd ~/code/perennial
make -j4 src/program_proof/simple/print_assumptions.vo
```

**This will take about 30 minutes.**

This only compiles the SimpleNFS top-level proof and all its dependencies,
including the GoJrnl proofs (in `src/program_proof/buftxn/sep_buftxn_proof.v`
and `sep_buftxn_recovery_proof.v`). The repository has a bunch of other research
code in it that isn't related to this paper.

We do proofs over Go using [Goose](https://github.com/tchajed/goose), which
compiles Go code to a Coq model. The output is checked in to the Perennial repo
for simplicity, but you can re-generate it from the goose-nfsd code:

```sh
cd ~/code/perennial
rm -r external/Goose/github_com/mit_pdos/goose_nfsd
./etc/update-goose.py --goose $GOOSE_PATH --nfsd $GOOSE_NFSD_PATH --skip-goose-examples --verbose
git status
```

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
the same, although if you don't have enough cores (or don't allocate them to the
VM) then performance will flatten at a smaller number of clients. Absolute
performance depends highly on your drive's performance.
