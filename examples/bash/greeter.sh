#!/bin/bash

# Copyright 2013 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# For each line FOO received on STDIN, respond with "Hello FOO!".
while read LINE
do
	echo "Hello $LINE!"
done