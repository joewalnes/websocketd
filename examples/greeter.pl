#!/usr/bin/perl

# For each line FOO received on STDIN, respond with "Hello FOO!".

# Autoflush output
use IO::Handle;
STDOUT->autoflush(1);

# Read STDIN
while (<>) {
  chomp; # remove \n
  $visitors++;
  print "Hello $_!\n";
}
