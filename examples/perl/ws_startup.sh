#!/bin/sh

#
# /usr/local/websocketd/ws_startup.sh
#

umask 000
cd /usr/local/websocketd

# Startup websocketd and listen to port 32080.  Run in background.
#  --port is the port to listen to
#  --staticdir is the directory to server html files from

# websocketd should be in your path

nohup websocketd --port=32080 --staticdir=/usr/local/websocketd ./ws_perl.pl &

