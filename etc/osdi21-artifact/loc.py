#!/usr/bin/env python3

import glob
import os
import pandas as pd


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
    os.chdir(os.environ["PERENNIAL_PATH"])
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
    return pd.DataFrame.from_records(
        data, columns=["Component", "Lines of Coq"]
    )


print(jrnl_cert_table().to_string(index=False))
