#!/usr/bin/perl

# Copyright 2013 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

use strict;

# Autoflush output
use IO::Handle;
STDOUT->autoflush(1);

use Time::HiRes qw(sleep);

# Simple example script that counts to 10 at ~2Hz, then stops.
for my $count (1 .. 10) {
	print "$count\n";
	sleep 0.5;
}
