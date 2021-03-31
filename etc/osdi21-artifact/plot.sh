#!/bin/bash

./bench.py data/bench-raw.txt
./scale.py data/scale-raw.txt
gnuplot bench.plot
gnuplot scale.plot
