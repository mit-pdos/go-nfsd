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
little over 2GB.

The VM was created by using vagrant, as specified in the
[Vagrantfile](Vagrantfile).
**The user account is `vagrant` with no password** (that is, the
empty password). The user account has sudo access without a password. The setup
consists of running [vm-init.sh](vm-init.sh) and [vm-setup.sh](vm-setup.sh).

You can launch the VM headless and then SSH to it. There's a port forwarding
rule vagrant sets up so that `ssh -p 10322 vagrant@localhost` should work,
without a password prompt.

The artifact's README and setup code are located at `~/go-nfsd/artifact`. The
README.md file there might be out-of-date by the time you read this; please run
`git pull` when you start, or follow the README on GitHub rather than in the VM.
Most of the work happens in `~/go-nfsd/eval`.

We've configured the VM with 8GB of RAM and 4 cores. You'll definitely want to
set it to the maximum number of cores you can afford for the scalability
experiment. Less RAM also might work but could lower performance.

## Claims

The artifact concerns four claims in the paper:

1. GoJournal's proof overhead is about 20x (in the tricky concurrent parts),
   while SimpleNFS is only 8x. Measured by lines of code.
2. The proofs for Perennial, GoJournal, and SimpleNFS are complete.
3. GoJournal is functional when compared against ext4 (using journaled data and
   over NFS for a fair comparison). We demonstrate this by showing GoNFS gets
   close throughput in the benchmarks in Figure 16, which use a RAM-backed disk.
4. GoJournal is scalable. We demonstrate this by showing performance for the
   smallfile benchmark scales with the number of clients, on an SSD (Figure 17).

# Detailed instructions

## Performance evaluation

We've cloned several repositories for you into the VM, most notably:

- https://github.com/mit-pdos/go-journal (located at `~/code/go-journal`)
  implements GoJournal on top of a disk. The `jrnl` package as the top-level
  API.
- https://github.com/mit-pdos/go-nfsd (located at `~/go-nfsd`): includes
  SimpleNFS and GoNFS. SimpleNFS is in `simple/`, and the binary for `GoNFS` is
  `cmd/go-nfsd`. The artifact is implemented with several scripts in `eval` in
  this repo.
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

Read `data/lines-of-code.txt` for the lines of code.

The exact performance results will vary depending on your machine, and differ
from the paper results because they are run in a VM.

You can get the figures out of the VM by running (from your host machine):

```sh
rsync -a -e 'ssh -p 10322' vagrant@localhost:./go-nfsd/eval/fig ./
```

The graphs there are:

- `bench.pdf` benchmarks run on a RAMdisk, where `app` clones and compiles the
  xv6 operating system
- `bench-ssd.pdf` same benchmarks but on an SSD (whatever the host disk is,
  actually).
- `largefile.pdf` just the largefile benchmark run on a variety of
  configurations (not shown in the paper).
- `scale.pdf` scalability of the smallfile benchmark with a varying number of
  clients, on an SSD.

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

## Under-documented features

These are features that we didn't have reviewers run but which we thought were
useful code to have.

[`./eval.sh`](eval.sh) just has all the commands in one file.

`./tests.sh` runs the fsstress and fsx-linux test suites from the Linux test
project (all the setup for these is included in `vm-setup.sh`).

The VM hosted on Zenodo was not compiled with the setup to run DFSCQ to save
space in the image, but you should be able to do this yourself. Run the commands
in [vm-setup.sh](vm-setup.sh) for DFSCQ (this takes about 5 minutes). Then run
`bench.sh` as above, and it will include results for FSCQ. None of the plots
include this data but after running `bench.py` you can easily look at the raw
data including FSCQ results with `column -t eval/data/bench.data`.
