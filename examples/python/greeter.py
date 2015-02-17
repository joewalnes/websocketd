#!/usr/bin/python

# Copyright 2013 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

from sys import stdin, stdout

# For each line FOO received on STDIN, respond with "Hello FOO!".
while True:
  line = stdin.readline().strip()
  print('Hello %s!' % line)
  stdout.flush() # Remember to flush
