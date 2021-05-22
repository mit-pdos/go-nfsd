#!/bin/bash

./bench.py data/bench-raw.txt
./scale.py data/scale-raw.txt
./bench.plot -o fig/bench.pdf
./bench.plot -ssd -o fig/bench-ssd.pdf
gnuplot scale.plot
gnuplot largefile.plot
