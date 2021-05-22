#!/bin/bash

set -eu

output=fig/bench.png
input=data/bench.data
ssd=false

while [[ "$#" -gt 0 ]]; do
	case "$1" in
	-i | --input)
		shift
		input="$1"
		shift
		;;
	-o | --output)
		shift
		output="$1"
		shift
		;;
	-ssd)
		ssd=true
		shift
		;;
	default)
		echo "unknown option $1" 1>&2
		exit 1
		;;
	esac
done

# shellcheck disable=SC2016
if [ "$ssd" = "true" ]; then
	line1='column("linux-ssd")/column("linux-ssd")'
	line2='column("gonfs-ssd")/column("linux-ssd")'
	label=" (SSD)"
	label1=$(awk 'NR==2 {printf "%.0f", $4}' "$input")
	label2=$(awk 'NR==3 {printf "%.0f", $4}' "$input")
	label3=$(awk 'NR==4 {printf "%.3f", $4}' "$input")
else
	label=""
	line1='column("linux")/column("linux")'
	line2='column("gonfs")/column("linux")'
	label1=$(awk 'NR==2 {printf "%.0f", $2}' "$input")
	label2=$(awk 'NR==3 {printf "%.0f", $2}' "$input")
	label3=$(awk 'NR==4 {printf "%0.3f", $2}' "$input")
fi

gnuplot <<-EOF
  set terminal png noenhanced size 1050,450
	set output "${output}"

	set style data histogram
	set style histogram cluster gap 1
	set rmargin at screen .95

	set xrange [-1:4]
	set yrange [0:1.45]
	set grid y
	set ylabel "Relative througput"
	set ytics scale 0.5,0 nomirror
	set xtics scale 0,0
	set key top right
	set style fill solid 1 border rgb "black"

	set label '${label1} file/s' at (0.15 -4./7),1.1 right rotate by 90 offset character 0,-1
	set label '${label2} MB/s' at (1.15 -4./7),1.1 right rotate by 90 offset character 0,-1
	set label '${label3} app/s' at (2.15 -4./7),1.1 right rotate by 90 offset character 0,-1

	set datafile separator "\t"

	plot "${input}" \
	        using (${line1}):xtic(1) title "Linux${label}" lc rgb '#b6d7a8' lt 1, \
	     '' using (${line2}):xtic(1) title "GoNFS${label}" lc rgb '#3a81ba' lt 1
EOF
