#!/usr/bin/env python3

import sys
import re

import argparse
import pandas as pd


def parse_raw(lines):
    fs = None
    data = []

    def get_bench_data(pattern, line):
        m = re.match(pattern, line)
        if m:
            return {
                "fs": fs,
                "bench": m.group("bench"),
                "val": float(m.group("val")),
            }
        return None

    for line in lines:
        m = re.match(r"""fs=(?P<fs>.*)""", line)
        if m:
            fs = m.group("fs")
            continue
        m = re.match(
            r"""fs-smallfile: (?P<clients>\d*) (?P<val>[0-9.]*) file/sec""",
            line,
        )
        if m:
            data.append(
                {
                    "fs": fs,
                    "clients": int(m.group("clients")),
                    "throughput": float(m.group("val")),
                }
            )
            continue
        print("ignored line: " + line, end="", file=sys.stderr)

    return pd.DataFrame.from_records(data)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("bench", type=argparse.FileType("r"))

    args = parser.parse_args()

    tidy_df = parse_raw(args.bench)
    df = tidy_df.pivot_table(index="clients", columns="fs", values="throughput")
    with open("data/gnfs.data", "w") as f:
        print(df["gonfs"].to_csv(sep="\t", header=False), end="", file=f)
    with open("data/linux-nfs.data", "w") as f:
        print(df["linux"].to_csv(sep="\t", header=False), end="", file=f)
