#!/bin/bash

# Simple example script that counts to 10 at ~2Hz, then stops.

for COUNT in $(seq 1 10)
do
  echo $COUNT
  sleep 0.5
done
