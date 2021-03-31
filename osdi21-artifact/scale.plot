set terminal png noenhanced size 1050,600
set output "fig/scale.png"

set auto x
set yrange [0:*]
set xtics 1
set ylabel "files / sec"
set format y '%.0s%c'
set xlabel "\# clients"
set key top left

set style data line

set style line 81 lt 0  # dashed
set style line 81 lt rgb "#808080"  # grey
set grid back linestyle 81

set border 3 back linestyle 80
set style line 1 lt rgb "#00A000" lw 2 pt 6
set style line 2 lt rgb "#5060D0" lw 2 pt 1
set style line 3 lt rgb "#A00000" lw 2 pt 2
set style line 4 lt rgb "#F25900" lw 2 pt 9

plot \
  "data/gnfs.data" using 1:($2) with linespoints ls 1 title 'GoNFS', \
  "data/linux-nfs.data" using 1:($2) with linespoints ls 2 title 'Linux NFS', \
  "data/serial.data" using 1:($2) with linespoints ls 3 title 'Serial GoNFS', \

# "fig/ext3.data" using 1:($2) with linespoints title 'Linux Ext3', \
