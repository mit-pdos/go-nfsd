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
        if re.match(r"""^#""", line):
            continue
        m = re.match(r"""fs=(?P<fs>.*)""", line)
        if m:
            fs = m.group("fs")
            continue
        item = get_bench_data(
            r"""fs-(?P<bench>smallfile): \d* (?P<val>[0-9.]*) file/sec""", line
        )
        if item:
            data.append(item)
            continue
        item = get_bench_data(
            r"""fs-(?P<bench>largefile):.* throughput (?P<val>[0-9.]*) MB/s""",
            line,
        )
        if item:
            data.append(item)
            continue
        item = get_bench_data(r"""(?P<bench>app)-bench (?P<val>[0-9.]*) app/s""", line)
        if item:
            data.append(item)
            continue
        print("ignored line: " + line, end="", file=sys.stderr)

    return pd.DataFrame.from_records(data)


def from_tidy(df):
    df = df.pivot_table(index="bench", columns="fs", values="val")
    df = df.reindex(index=["smallfile", "largefile", "app"])
    return df


def largefile_from_tidy(df):
    df = df[df["bench"] == "largefile"]
    df = df.pivot_table(index="fs", columns="bench", values="val")
    return df


def main():
    from os.path import join

    parser = argparse.ArgumentParser()

    parser.add_argument(
        "-o",
        "--output",
        default="data",
        help="directory to output bench.data and largefile.data to",
    )
    parser.add_argument(
        "bench", type=argparse.FileType("r"), help="raw output from bench.sh"
    )

    args = parser.parse_args()

    tidy_df = parse_raw(args.bench)
    df = from_tidy(tidy_df)
    # list out columns again to get order right
    columns = ["linux", "gonfs"]
    if "linux-ssd" in df.columns:
        columns.extend(["linux-ssd", "gonfs-ssd"])
    df.to_csv(join(args.output, "bench.data"), sep="\t", columns=columns),

    df = largefile_from_tidy(tidy_df)
    df.to_csv(join(args.output, "largefile.data"), sep="\t"),


if __name__ == "__main__":
    main()
