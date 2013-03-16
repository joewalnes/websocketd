#!/usr/bin/ruby

# Copyright 2013 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Autoflush output
STDOUT.sync = true

# For each line FOO received on STDIN, respond with "Hello FOO!".
while 1
  line = STDIN.readline.strip
  puts "Hello #{line}!"
end
