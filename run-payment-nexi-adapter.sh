#! /bin/bash

STARTTIME=$(date '+%Y-%m-%d_%H-%M-%S')

echo "Writing log to ~/work/logs/payment-nexi-adapter.$STARTTIME.log"
echo "Send Ctrl-C/SIGTERM to initiate graceful shutdown"

cd ~/work/payment-nexi-adapter

./payment-nexi-adapter -config config.yaml &> ~/work/logs/payment-nexi-adapter.$STARTTIME.log

