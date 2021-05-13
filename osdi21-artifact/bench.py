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


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("bench", type=argparse.FileType("r"))

    args = parser.parse_args()

    tidy_df = parse_raw(args.bench)
    df = tidy_df.pivot_table(index="bench", columns="fs", values="val")
    df = df.reindex(index=["smallfile", "largefile", "app"])
    df.rename(columns={"linux": "Linux", "gonfs": "GoNFS"}, inplace=True)
    with open("data/bench.data", "w") as f:
        print(df.to_csv(sep="\t", columns=["linux", "gonfs"]), end="", file=f)
