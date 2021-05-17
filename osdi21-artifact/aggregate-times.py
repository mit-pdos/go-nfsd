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

import sys
import re
import collections

counts = collections.defaultdict(int)
totals = collections.defaultdict(float)

for line in sys.stdin:
    m = re.match(r"""(?P<proc>.*)\t(?P<time>.*)""", line)
    if m:
        proc = int(m.group("proc"))
        time_s = float(m.group("time"))
        counts[proc] += 1
        totals[proc] += time_s

proc_mapping = {
    0: "NULL",
    1: "GETATTR",
    3: "LOOKUP",
    4: "ACCESS",
    7: "WRITE",
    8: "CREATE",
    9: "MKDIR",
    12: "REMOVE",
    19: "FSINFO",
    20: "PATHCONF",
}

for proc in counts:
    proc_name = proc_mapping[proc] if proc in proc_mapping else str(proc)
    count = counts[proc]
    total_s = totals[proc]
    micros_per_op = total_s / count * 1e6
    print(f"{proc_name:>10}\t{count:8}\t{micros_per_op:.1f} us/op")
