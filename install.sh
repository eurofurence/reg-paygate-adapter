#! /bin/bash

set -o errexit

if [[ "$RUNTIME_USER" == "" ]]; then
  echo "RUNTIME_USER not set, bailing out. Please run setup.sh first."
  exit 1
fi

mkdir -p tmp
cp payment-nexi-adapter tmp/
cp config.yaml tmp/
cp run-payment-nexi-adapter.sh tmp/

chgrp $RUNTIME_USER tmp/*
chmod 640 tmp/config.yaml
chmod 750 tmp/payment-nexi-adapter
chmod 750 tmp/payment-nexi-adapter.sh
mv tmp/payment-nexi-adapter /home/$RUNTIME_USER/work/payment-nexi-adapter/
mv tmp/config.yaml /home/$RUNTIME_USER/work/payment-nexi-adapter/
mv tmp/run-payment-nexi-adapter.sh /home/$RUNTIME_USER/work/
