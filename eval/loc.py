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


def perennial_table():
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
        fstxn/*.go inode/*.go shrinker/shrinker.go super/super.go""".split()
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
    # converted to proper pandas missing records at the end, then printed as "---"

    def ratio(n, m):
        if m == 0:
            return -1
        return int(round(float(n) / m))

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
            entry("obj", txn_c, txn_p),
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


def array_to_latex_table(rows):
    latex_rows = [" & ".join(str(x) for x in row) for row in rows]
    return " \\\\ \n".join(latex_rows)


def loc(x):
    return "\\loc{" + str(x) + "}"


def perennial_to_latex(df):
    rows = []
    for _, row in df.iterrows():
        rows.append([row[0], loc(row[1])])
    return array_to_latex_table(rows)


def get_multirow(df, index, col, f):
    x = df.iloc[index, col]
    if index + 1 < len(df) and df.iloc[index + 1, col] == "---":
        return "\\multirow{2}{*}{" + str(f(x)) + "}"
    if x == "---":
        return ""
    return f(x)


def impl_to_latex(df):
    # set GoNFS lines of code to this text
    df.iloc[len(df) - 2, 2] = "Not verified"
    rows = []
    for index, row in df.iterrows():
        layer = row[0]
        if layer.islower():
            layer = "\\textsc{" + layer + "}"
        lines_c = row[1]
        lines_p = row[2]
        # total hack to fix last two lines
        if index < len(df) - 3:
            ratio = get_multirow(df, index, 3, lambda x: x)
        else:
            ratio = row[3]
        rows.append([layer, lines_c, lines_p, ratio])
    return array_to_latex_table(rows)


if __name__ == "__main__":
    import argparse
    from os.path import join

    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--latex",
        help="output LaTeX table to this directory",
        default=None,
    )

    args = parser.parse_args()

    original_pwd = os.getcwd()

    perennial_df = perennial_table()
    impl_df = program_proof_table().fillna("---")

    os.chdir(original_pwd)

    if args.latex is None:
        print("Lines of code in Perennial")
        print(perennial_df.to_string(index=False))
        print()

        print("Lines of code for GoJournal and SimpleNFS")
        print(impl_df.to_string(index=False))
    else:
        with open(join(args.latex, "perennial-loc.tex"), "w") as f:
            print(perennial_to_latex(perennial_df), file=f)
        with open(join(args.latex, "impl-loc.tex"), "w") as f:
            print(impl_to_latex(impl_df), file=f)
