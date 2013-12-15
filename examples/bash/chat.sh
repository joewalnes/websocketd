#!/bin/bash

# Copyright 2013 Jeroen Janssens
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Run a simple chat server: websocketd --devconsole --port 8080 ./chat.sh
#
# Please note that this example requires GNU tail, which is not the default
# tail on OS X. Even though this script properly escapes the variables,
# please keep in mind that it is in general a bad idea to read
# untrusted data into variables and pass this onto the command line.

echo "Please enter your name:"; read USER
echo "[$(date)] ${USER} joined the chat" >> chat.log
echo "[$(date)] Welcome to the chat ${USER}!"
tail -n 0 -f chat.log --pid=$$ | grep --line-buffered -v "] ${USER}>" &
while read MSG; do echo "[$(date)] ${USER}> ${MSG}" >> chat.log; done
