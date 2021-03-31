set terminal png noenhanced size 1050,450
set output "fig/bench.png"

set style data histogram
set style histogram cluster gap 1
set rmargin at screen .95

set xrange [-1:4.5]
set yrange [0:*]
set grid y
set ylabel "Relative througput"
set ytics scale 0.5,0 nomirror
set xtics scale 0,0
set key top right
set style fill solid 1 border rgb "black"

set label '3141.3 file/s' at (0.15 -4./7),1 right rotate by 90 offset character 0,-1
set label '199.25 MB/s' at (1.15 -4./7),1 right rotate by 90 offset character 0,-1
set label '0.617 app/s' at (2.15 -4./7),1 right rotate by 90 offset character 0,-1

plot "data/bench.data" \
        using ($2/$2):xtic(1) title col lc rgb '#b6d7a8' lt 1, \
     '' using ($3/$2):xtic(1) title col lc rgb '#3a81ba' lt 1, \
