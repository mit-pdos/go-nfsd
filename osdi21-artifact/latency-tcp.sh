#!/bin/bash

# run various performance benchmarks

set -eu

blue=$(tput setaf 4)
red=$(tput setaf 1)
reset=$(tput sgr0)

info() {
  echo -e "${blue}$1${reset}" 1>&2
}
error() {
  echo -e "${red}$1${reset}" 1>&2
}

usage() {
  echo "Usage: $0 [-ssd <block device or file path>]" 1>&2
  echo "SSD benchmarks will be skipped if -ssd is not passed"
}

ssd_file=""

while [[ "$#" -gt 0 ]]; do
    case "$1" in
        -ssd)
            shift
            ssd_file="$1"
            shift
            ;;
        *)
            error "unexpected argument $1"
            usage
            exit 1
            ;;
    esac
done

if [ ! -d "$GOOSE_NFSD_PATH" ]; then
    echo "GOOSE_NFSD_PATH is unset" 1>&2
    exit 1
fi

cd "$GOOSE_NFSD_PATH"

DATA_PATH=$GOOSE_NFSD_PATH/osdi21-artifact/data
#chmod 777 $DATA_PATH
TMP=/tmp

info "GoNFS (smallfile)"
echo "#GoNFS (smallfile)" > $DATA_PATH/gonfs-latencies-tcp.txt
sudo tshark -i lo -f tcp -w $TMP/gonfs-smallfile.pcap &
./bench/run-goose-nfs.sh -stats true -unstable=false -disk "" go run ./cmd/fs-smallfile -benchtime=20s
sleep 1
sudo killall tshark
sleep 1
sudo tshark -Tfields -e 'nfs.procedure_v3' -e 'rpc.time' -r $TMP/gonfs-smallfile.pcap '(nfs && rpc.time)' | ./osdi21-artifact/aggregate-times.py | tee -a $DATA_PATH/gonfs-latencies-tcp.txt

info "GoNFS (null)"
echo "#GoNFS (null)" >> $DATA_PATH/gonfs-latencies-tcp.txt
sudo tshark -i lo -f tcp -w $TMP/gonfs-null.pcap &
./bench/run-goose-nfs.sh -stats true -unstable=false -disk "" go run ./cmd/clnt-null -benchtime=20s
sleep 1
sudo killall tshark
sleep 1
sudo tshark -Tfields -e 'nfs.procedure_v3' -e 'rpc.time' -r $TMP/gonfs-null.pcap '(nfs && rpc.time)' | ./osdi21-artifact/aggregate-times.py | tee -a $DATA_PATH/gonfs-latencies-tcp.txt

info "\n\nResults: "
cat osdi21-artifact/data/gonfs-latencies-tcp.txt

echo 1>&2
info "Linux ext4 over NFS"
echo "#Linuxt ext4 over NFS (smallfile)" > $DATA_PATH/linux-latencies-tcp.txt
sudo tshark -i lo -f tcp -w $TMP/linux-smallfile.pcap &
./bench/run-linux.sh go run ./cmd/fs-smallfile -benchtime=20s
sleep 1
sudo killall tshark
sleep 1
sudo tshark -Tfields -e 'nfs.procedure_v3' -e 'rpc.time' -r $TMP/linux-smallfile.pcap '(nfs && rpc.time)' | ./osdi21-artifact/aggregate-times.py | tee -a $DATA_PATH/linux-latencies-tcp.txt


echo 1>&2
info "Linux ext4 over NFS (null)"
echo "#Linuxt ext4 over NFS (null)" >> $DATA_PATH/linux-latencies-tcp.txt
sudo tshark -i lo -f tcp -w $TMP/linux-null.pcap &
./bench/run-linux.sh go run ./cmd/clnt-null -benchtime=20s 
sleep 1
sudo killall tshark
sleep 1
sudo tshark -Tfields -e 'nfs.procedure_v3' -e 'rpc.time' -r $TMP/linux-null.pcap '(nfs && rpc.time)' | ./osdi21-artifact/aggregate-times.py | tee -a $DATA_PATH/linux-latencies-tcp.txt

info "\n\nResults: "
cat osdi21-artifact/data/linux-latencies-tcp.txt
