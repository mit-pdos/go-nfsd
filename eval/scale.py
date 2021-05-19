#!/usr/bin/env python3

import sys
import re
from os.path import join

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
        if re.match(r"""^#""", line):
            continue
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


def from_tidy(tidy_df):
    df = tidy_df.pivot_table(index="clients", columns="fs", values="throughput")
    # propagate serial results forward
    df.fillna(method="ffill", inplace=True)
    return df


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "-o",
        "--output",
        default="data",
        help="directory to output *.data files to",
    )
    parser.add_argument(
        "bench", type=argparse.FileType("r"), help="path to scale.sh raw output"
    )

    args = parser.parse_args()

    df = from_tidy(parse_raw(args.bench))
    with open(join(args.output, "gnfs.data"), "w") as f:
        print(df["gonfs"].to_csv(sep="\t", header=False), end="", file=f)
    with open(join(args.output, "linux-nfs.data"), "w") as f:
        print(df["linux"].to_csv(sep="\t", header=False), end="", file=f)
    with open(join(args.output, "serial.data"), "w") as f:
        print(df["serial-gonfs"].to_csv(sep="\t", header=False), end="", file=f)


if __name__ == "__main__":
    main()
