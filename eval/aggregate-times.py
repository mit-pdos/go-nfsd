#!/usr/bin/env python3

# aggregate the results of running tshark over an NFS packet capture
#
# gather data with
# tshark -i lo -f tcp -w nfs.pcap
#
# then process with
# tshark -Tfields -e 'nfs.procedure_v3' -e 'rpc.time' -r nfs.pcap '(nfs && rpc.time)' | ./aggregate-times.py
#
# note that running tshark over a trace takes a while

import re
import numpy as np

proc_mapping = {
    0: "NULL",
    1: "GETATTR",
    2: "SETATTR",
    3: "LOOKUP",
    4: "ACCESS",
    6: "READ",
    7: "WRITE",
    8: "CREATE",
    9: "MKDIR",
    10: "SYMLINK",
    12: "REMOVE",
    13: "RMDIR",
    14: "RENAME",
    15: "LINK",
    16: "READDIR",
    17: "READDIRPLUS",
    18: "FSSTAT",
    19: "FSINFO",
    20: "PATHCONF",
    21: "COMMIT",
}


def proc_latencies(f):
    latencies_s = {}
    for line in f:
        m = re.match(r"""(?P<proc>.*)\t(?P<time>.*)""", line)
        if m:
            # rarely tshark will put two timings on the same line, just ignore them
            if "," in m.group("proc"):
                continue
            proc = int(m.group("proc"))
            time_s = float(m.group("time"))
            if proc not in latencies_s:
                latencies_s[proc] = []
            latencies_s[proc].append(time_s)
    data = {}
    for proc, latencies in latencies_s.items():
        proc_name = proc_mapping[proc] if proc in proc_mapping else str(proc)
        latencies_us = np.array(latencies) * 1e6
        data[proc_name] = latencies_us
    return data


def main():
    import sys
    import argparse

    parser = argparse.ArgumentParser()

    parser.add_argument("-i", "--input", type=argparse.FileType("r"), default=sys.stdin)
    parser.add_argument(
        "--stats", action="store_true", help="report median and 90th percentile"
    )

    args = parser.parse_args()

    latencies_s = proc_latencies(args.input)

    for proc, latencies in latencies_s.items():
        count = latencies.size
        avg_micros = np.mean(latencies)
        print(f"{proc:>10}\t{count:8}\t{avg_micros:.1f} us/op\t", end="")
        if args.stats:
            print(
                f"(50th: {np.quantile(latencies, 0.5):0.1f} us)\t"
                + f"(90th: {np.quantile(latencies, 0.9):0.1f} us)",
                end="",
            )
        print()


if __name__ == "__main__":
    main()
