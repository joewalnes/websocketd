#!/usr/bin/python

# Copyright 2013 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

from sys import stdout
from time import sleep

# Simple example script that counts to 10 at ~2Hz, then stops.
for count in range(0, 10):
  print(count + 1)
  stdout.flush() # Remember to flush
  sleep(0.5)
