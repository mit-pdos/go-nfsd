# JrnlCert artifact

## Getting started

Read the description below on the VM. You should be able to run all the commands
in this artifact quickly with the exception of compiling Perennial, so to get
started we recommend gathering all the data and skipping the compilation step.
Comparing the results and compiling the proofs in Perennial will take a bit more
time.

### About the VM

The VM was created by installing the Ubuntu 20.04 live server image in
VirtualBox. **The user account is `ubuntu` with password `ubuntu`**. The user
account has sudo access without a password.

You can launch the VM headless and then SSH to it. There's a port forwarding
rule set up in VirtualBox so that `ssh -p 10322 ubuntu@localhost` should work.
You might want to add your public key to the VM to avoid having to type the
password every time.

The artifact (including this README) is located at
`~/goose-nfsd/osdi21-artifact`. The README.md file there might be out-of-date by
the time you read this; please run `git pull` when you start, or follow the
README on GitHub rather than in the VM.

We've configured the VM with 8GB of RAM and 6 cores. You'll definitely want to
set it to the maximum number of cores you can afford for the scalability
experiment. Less RAM also might work but could lower performance.

## Claims

The artifact concerns four claims in the paper:

1. GoJrnl's proof overhead is about 20x (in the tricky concurrent parts), while
   SimpleNFS is only 7x. Measured by lines of code.
2. The proofs for JrnlCert, GoJrnl, and SimpleNFS are complete.
3. GoJrnl is functional when compared against ext3 (using journaled data and
   over NFS for a fair comparison). We demonstrate this by showing GoNFS gets
   close throughput in the benchmarks in Figure 16, which use a RAM-backed disk.
4. GoJrnl is scalable. We demonstrate this by showing performance for the
   smallfile benchmark scales with the number of clients, on an SSD (Figure 17).

## Performance evaluation

We've cloned several repositories for you into the VM, most notably:

- https://github.com/mit-pdos/goose-nfsd (located at `~/goose-nfsd`): includes
  GoJrnl, SimpleNFS, and GoNFS. The journal is implemented by the `buftxn`
  package, SimpleNFS is in `simple/`, and the binary for `GoNFS` is
  `cmd/goose-nfsd` (which imports various packages in this repo). The artifact
  is implemented with several scripts in `osdi21-artifact` in this repo.
- https://github.com/mit-pdos/perennial (located at `~/perennial`): the Perennial framework (renamed
  JrnlCert for submission) and all program proofs for the journal and SimpleNFS.

### Gather data

This should all be done in the artifact directory:

```sh
cd ~/goose-nfsd/osdi21-artifact
```

```sh
./loc.py | tee data/lines-of-code.txt
```

Instantaneous. The numbers won't match up exactly with the paper (see below
under "Check output").

```sh
./bench.sh | tee data/bench-raw.txt
```

Takes about a minute. You can manually inspect the output file (which is fairly
readable) if you'd like.

```sh
./scale.sh 10 | tee data/scale-raw.txt
```

**Takes a few minutes** (the 10 is the number of clients to run till; you can use a
smaller number of you want it to finish faster).

### Produce graphs

```sh
./plot.sh
```

(assumes you've put data in `data/bench-raw.txt` and `data/scale-raw.txt`, as
above)

If you haven't run `./scale.sh` yet, then you can still generate the benchmark
figure with `./bench.py data/bench-raw.txt && gnuplot bench.plot`.

### Check output

Compare `data/lines-of-code.txt` to Figures 14 and 15 in the paper. The exact
lines won't be the same because the code has changed slightly (and this artifact
is automated differently from how the original data was gathered), but the
numbers should generally line up and the overall conclusion about proof overhead
still holds.

The exact performance results will vary depending on your machine, and suffer
slightly from being run in a VM.

You can get the figures out of the VM by running (from your host machine):

```sh
rsync -a -e 'ssh -p 10322' ubuntu@localhost:./goose-nfsd/osdi21-artifact/fig ./
```

Compare `fig/bench.png` to Figure 16 in the paper. The absolute performance
numbers were included manually in the graph; you can easily find the numbers by
looking at `data/bench.data` and looking at the "linux" column. This benchmark
seems to reproduce poorly in a VM, particularly the largefile workload.

Compare `fig/scale.png` to Figure 17 in the paper. The scaling should be roughly
the same, although if you don't have enough cores (or don't allocate them to the
VM) then performance will flatten at a smaller number of clients. Absolute
performance depends highly on your drive's performance.

## Compiling the proofs

The paper claims to have verified the GoJrnl implementation. You should check
this by compiling the proofs in the `perennial` repo:

```sh
cd ~/perennial
make -j4 src/program_proof/simple/print_assumptions.vo
```

**This will take 30-40 minutes, and 60-70 CPU minutes.**

This only compiles the SimpleNFS top-level proof and all its dependencies,
including the GoJrnl proofs (in `src/program_proof/buftxn/sep_buftxn_proof.v`
and `sep_buftxn_recovery_proof.v`). The repository has a bunch of other research
code in it that isn't related to this paper.

We do proofs over Go using [Goose](https://github.com/tchajed/goose), which
compiles Go code to a Coq model. The output is checked in to the Perennial repo
for simplicity, but you can re-generate it from the goose-nfsd code:

```sh
cd ~/perennial
rm -r external/Goose/github_com/mit_pdos/goose_nfsd
./etc/update-goose.py --goose $GOOSE_PATH --nfsd $GOOSE_NFSD_PATH --skip-goose-examples --verbose
git status
```

The final `git status` command should report that the working tree has been
restored to its previous state.
