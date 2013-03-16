#!/usr/bin/ruby

# Copyright 2013 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Autoflush output
STDOUT.sync = true

# Simple example script that counts to 10 at ~2Hz, then stops.
(1..10).each do |count| 
	puts count
	sleep(0.5)
end
