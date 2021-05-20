# GoJournal: a verified, concurrent, crash-safe journaling system (Artifact)

[![License: CC BY
4.0](https://img.shields.io/badge/License-CC%20BY%204.0-lightgrey.svg)](https://creativecommons.org/licenses/by/4.0/)

The text of this artifact is licensed under the Creative Commons Attribution 4.0
license. The code is under the same MIT license as the parent repo.

# Getting started

Read the description below on downloading and using the VM. You should be able
to run all the commands in this artifact quickly (within 30 minutes) with the
exception of compiling Perennial, so to get started we recommend gathering all
the data and skipping the compilation step. Comparing the results and compiling
the proofs in Perennial will take a bit more time.

### About the VM

You can get the VM from Zenodo via DOI
[10.5281/zenodo.4657116](https://zenodo.org/record/4657115). The download is a
little over 3GB.

The VM was created by using vagrant, as specified in the
[Vagrantfile](Vagrantfile).
**The user account is `vagrant` with no password** (that is, the
empty password). The user account has sudo access without a password. After some
basic setup, like installing ZSH, we ran [vm-setup.sh](vm-setup.sh).

You can launch the VM headless and then SSH to it. There's a port forwarding
rule vagrant sets up so that `ssh -p 10322 vagrant@localhost` should work,
without a password prompt.

The artifact (including this README) is located at
`~/go-nfsd/artifact`. The README.md file there might be out-of-date by
the time you read this; please run `git pull` when you start, or follow the
README on GitHub rather than in the VM.

We've configured the VM with 8GB of RAM and 6 cores. You'll definitely want to
set it to the maximum number of cores you can afford for the scalability
experiment. Less RAM also might work but could lower performance.

## Claims

The artifact concerns four claims in the paper:

1. GoJournal's proof overhead is about 20x (in the tricky concurrent parts),
   while SimpleNFS is only 8x. Measured by lines of code.
2. The proofs for Perennial, GoJournal, and SimpleNFS are complete.
3. GoJournal is functional when compared against ext3 (using journaled data and
   over NFS for a fair comparison). We demonstrate this by showing GoNFS gets
   close throughput in the benchmarks in Figure 16, which use a RAM-backed disk.
4. GoJournal is scalable. We demonstrate this by showing performance for the
   smallfile benchmark scales with the number of clients, on an SSD (Figure 17).

# Detailed instructions

## Performance evaluation

We've cloned several repositories for you into the VM, most notably:

- https://github.com/mit-pdos/go-journal (located at `~/go-journal`) implements
  GoJournal on top of a disk. The `jrnl` package as the top-level API.
- https://github.com/mit-pdos/go-nfsd (located at `~/go-nfsd`): includes
  SimpleNFS and GoNFS. SimpleNFS is in `simple/`, and the binary for `GoNFS` is
  `cmd/go-nfsd` (which imports various packages in this repo). The artifact
  is implemented with several scripts in `eval` in this repo.
- https://github.com/mit-pdos/perennial (located at `~/perennial`): the
  Perennial framework and all program proofs for GoJournal and SimpleNFS.

### Gather data

This should all be done in the eval directory:

```sh
cd ~/go-nfsd/eval
```

```sh
./loc.py | tee data/lines-of-code.txt
```

Instantaneous. The numbers won't match up exactly with the paper (see below
under "Check output").

```sh
./bench.sh -ssd ~/disk.img
```

Takes 2-3 minutes. You can manually inspect the output file at
`eval/data/bench-raw.txt` (which is fairly readable) if you'd like.

```sh
./scale.sh 10
```

**Takes a few minutes** (the 10 is the number of clients to run till; you can use a
smaller number of you want it to finish faster). Outputs to `eval/data/scale-raw.txt`.

### Produce graphs

```sh
./plot.sh
```

If you haven't run `./scale.sh` yet, then you can still generate the benchmark
figure with `./bench.py data/bench-raw.txt && gnuplot bench.plot`.

### Check output

Compare `data/lines-of-code.txt` to Figures 13 and 14 in the paper.

The exact performance results will vary depending on your machine, and suffer
slightly from being run in a VM.

You can get the figures out of the VM by running (from your host machine):

```sh
rsync -a -e 'ssh -p 10322' vagrant@localhost:./go-nfsd/eval/fig ./
```

Compare `fig/bench.png` to Figure 16 in the paper. The absolute performance
numbers were included manually in the graph; you can easily find the numbers by
looking at `data/bench.data` and looking at the "linux" column. We made a
mistake and misconfigured this benchmark; with the fixed configuration, both
file systems perform better and GoNFS gets better performance than Linux. This
seems to be because it is able to take advantage of a much larger batch write
size than the Linux NFS server. For the camera-ready version of the paper we
already plan to expand the evaluation to better explain the performance on this
microbenchmark (we may still be running Linux in a way that gets lower
performance).

Compare `fig/scale.png` to Figure 17 in the paper. The scaling should be roughly
the same, although if you don't have enough cores (or don't allocate them to the
VM) then performance will flatten at a smaller number of clients. Absolute
performance depends highly on your drive's performance.

## Compile the proofs

The paper claims to have verified the GoJournal implementation. You should check
this by compiling the proofs in the `perennial` repo:

```sh
cd ~/perennial
make -j4 src/program_proof/simple/print_assumptions.vo
```

**This will take 30-40 minutes, and 60-70 CPU minutes.**

This only compiles the SimpleNFS top-level proof and all its dependencies,
including the GoJournal proofs (in `src/program_proof/buftxn/sep_buftxn_proof.v`
and `sep_buftxn_recovery_proof.v`). The repository has a bunch of other research
code in it that isn't related to this paper.

We do proofs over Go using [Goose](https://github.com/tchajed/goose), which
compiles Go code to a Coq model. The output is checked in to the Perennial repo
for simplicity, but you can re-generate it from the go-nfsd code:

```sh
cd ~/perennial
rm -r external/Goose/github_com/mit_pdos/go_journal
rm -r external/Goose/github_com/mit_pdos/go_nfsd
./etc/update-goose.py --goose $GOOSE_PATH --journal $GO_JOURNAL_PATH --nfsd $GO_NFSD_PATH --skip-goose-examples --verbose
git status
```

The final `git status` command should report that the working tree has been
restored to its previous state.

## Undocumented features

These are features that we didn't think were important for evaluation but might
be useful for our own reference.

[`./eval.sh`](eval.sh) runs the whole evaluation, including plotting

`./tests.sh` runs the fsstress and fsx-linux test suites from the Linux test
project (all the setup for these is included in `vm-setup.sh`).
