#!/bin/bash

./bench.py data/bench-raw.txt
./scale.py data/scale-raw.txt
./bench.plot
gnuplot scale.plot
gnuplot largefile.plot
