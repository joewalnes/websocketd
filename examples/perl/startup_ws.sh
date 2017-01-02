#!/bin/sh

cd /usr/local/bin/websocketd

umask 000
nohup ./websocketd --port=32080 --staticdir=/usr/local/bin/websocketd ./perl_ws.pl &

