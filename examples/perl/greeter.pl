#!/usr/bin/perl

# Copyright 2013 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

use strict;

# Autoflush output
use IO::Handle;
STDOUT->autoflush(1);

# For each line FOO received on STDIN, respond with "Hello FOO!".
while (<>) {
  chomp; # remove \n
  print "Hello $_!\n";
}
