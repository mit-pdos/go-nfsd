set terminal pdf dashed noenhanced size 2.5in,1.5in
set output "fig/largefile.pdf"

set style data histogram
set style histogram cluster gap 1
set rmargin at screen .95

set xrange [-1:4]
set yrange [0:*]
set grid y
set ylabel "Throughput (MB/s)"
set ytics scale 0.5,0 nomirror
set xtics scale 0,0
set key top right
set style fill solid 1 border rgb "black"

set datafile separator "\t"

plot "data/largefile.data" \
        using "linux-ssd":xtic(1) title "Linux (data=journal)" lc rgb '#b6d7a8' lt 1, \
        '' using "gonfs-ssd-unstable":xtic(1) title "GoNFS (unstable)" lc rgb '#3a81ba' lt 1, \
        '' using "linux-ssd-sync":xtic(1) title "Linux (sync)" lc rgb "#e4ffd9" lt 1, \
        '' using "gonfs-ssd":xtic(1) title "GoNFS (stable)" lc rgb "#8dbbe0" lt 1
