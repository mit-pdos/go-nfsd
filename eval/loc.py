#!/usr/bin/env python3

# Produce data for figures 14 and 15 (lines of code).
#
# To run this script, set PERENNIAL_PATH, GO_NFSD_PATH, and MARSHAL_PATH to
# checkouts of those three projects.

import glob
import os
import numpy as np
import pandas as pd


def goto_path(var_prefix):
    assert var_prefix in ["perennial", "go_nfsd", "go_journal", "marshal"]
    os.chdir(os.environ[var_prefix.upper() + "_PATH"])


def count_lines_file(p):
    """Return the number of lines in file at path p."""
    with open(p) as f:
        return sum(1 for _ in f)


def count_lines_pattern(pat):
    return sum(count_lines_file(fname) for fname in glob.glob(pat))


def wc_l(*patterns):
    return sum(count_lines_pattern(pat) for pat in patterns)


def jrnl_cert_table():
    """Generate figure 14 (lines of code for JrnlCert)"""
    goto_path("perennial")
    helpers = wc_l(
        "src/Helpers/*.v",
        "src/iris_lib/*.v",
        "src/algebra/big_op/*.v",
        "src/algebra/liftable.v",
    )
    ghost_state = wc_l("src/algebra/*.v") - wc_l("src/algebra/liftable.v")
    program_logic = wc_l("src/program_logic/*.v")
    data = [
        (
            "Helper libraries (maps, lifting, tactics)",
            helpers,
        ),
        (
            "Ghost state and resources",
            ghost_state,
        ),
        (
            "Program logic for crashes",
            program_logic,
        ),
        (
            "Total",
            helpers + ghost_state + program_logic,
        ),
    ]
    return pd.DataFrame.from_records(data, columns=["Component", "Lines of Coq"])


def program_proof_table():
    """Generate figure 15 (lines of code for GoJournal and SimpleNFS)"""

    # get all lines of code from go-journal
    goto_path("go_journal")
    circ_c = wc_l("wal/0circular.go")
    wal_c = wc_l("wal/*.go") - circ_c - wc_l("wal/*_test.go")
    txn_c = wc_l("obj/obj.go")
    jrnl_c = wc_l("jrnl/jrnl.go")
    lockmap_c = wc_l("lockmap/lock.go")
    misc_c = wc_l("addr/addr.go", "buf/buf.go", "buf/bufmap.go")
    goto_path("marshal")
    misc_c += wc_l("marshal.go")
    goto_path("go_nfsd")
    go_nfs_c = wc_l(
        *"""nfs/*.go alloc/alloc.go
        alloctxn/alloctxn.go cache/cache.go cmd/go-nfsd/main.go
        common/common.go dcache/dcache.go dir/dir.go dir/dcache.go fh/nfs_fh.go
        fstxn/*.go inode/* shrinker/shrinker.go super/super.go
        txn/txn.go util/util.go""".split()
    )
    simple_c = wc_l("simple/0super.go", "simple/fh.go", "simple/ops.go")

    # get all lines of proof from Perennial
    goto_path("perennial")
    os.chdir("src/program_proof")
    circ_p = wc_l("wal/circ_proof*.v")
    wal_heapspec_p = wc_l("wal/heapspec.v") + wc_l("wal/heapspec_lib.v")
    wal_p = (
        wc_l("wal/*.v")
        # don't double-count
        - circ_p
        - wal_heapspec_p
        # just an experiment, not used
        - wc_l("wal/heapspec_list.v")
    )
    txn_p = wc_l("txn/*.v")
    jrnl_p = wc_l("buftxn/buftxn_proof.v")
    sep_jrnl_p = wc_l("buftxn/sep_buftxn_*.v")
    lockmap_p = wc_l("*lockmap_proof.v")
    misc_p = wc_l(
        "addr/*.v",
        "buf/*.v",
        "disk_lib.v",
        "marshal_block.v",
        "marshal_proof.v",
        "util_proof.v",
    )
    simple_p = wc_l("simple/*.v")

    # note that the table uses -1 as a sentinel for missing data; these are
    # converted to proper pandas missing records at the end, then printed as "-"

    def ratio(n, m):
        if m == 0:
            return -1
        return int(float(n) / m)

    def entry(name, code, proof):
        return (name, code, proof, ratio(proof, code))

    def entry_nocode(name, proof):
        return (name, -1, proof, -1)

    schema = [
        ("layer", "U25"),
        ("Lines of code", "i8"),
        ("Lines of proof", "i8"),
        ("Ratio", "i8"),
    ]

    data = np.array(
        [
            entry("circular", circ_c, circ_p),
            ("wal-sts", wal_c, wal_p, ratio(wal_p + wal_heapspec_p, wal_c)),
            entry_nocode("wal", wal_heapspec_p),
            entry("txn", txn_c, txn_p),
            (
                "jrnl-sts",
                jrnl_c,
                jrnl_p,
                ratio(jrnl_p + sep_jrnl_p, jrnl_c),
            ),
            entry_nocode("jrnl", sep_jrnl_p),
            entry("lockmap", lockmap_c, lockmap_p),
            entry("Misc.", misc_c, misc_p),
        ],
        dtype=schema,
    )
    df = pd.DataFrame.from_records(data)
    total_c = df["Lines of code"].sum()
    total_p = df["Lines of proof"].sum()
    df = df.append(
        pd.DataFrame.from_records(
            np.array(
                [
                    entry("GoJournal total", total_c, total_p),
                    ("GoNFS", go_nfs_c, -1, -1),
                    entry("SimpleNFS", simple_c, simple_p),
                ],
                dtype=schema,
            )
        ),
        ignore_index=True,
    ).replace(-1, pd.NA)
    return df


print("~ Fig 14 (lines of code in Perennial)")
print(jrnl_cert_table().to_string(index=False))
print()

print("~ Fig 15 (lines of code for GoJournal and SimpleNFS)")
print(program_proof_table().fillna("-").to_string(index=False))
