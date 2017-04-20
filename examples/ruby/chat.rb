#!/usr/bin/env ruby

# A ruby based websocketd chat server.
#
# Copyright 2013 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.
#
# Requirements:
# - redis (run `redis-cli ping` to check if installed)
# - redis gem (run `gem install redis`)
#
# Usage:
# Start the server:
#
# $ websocketd --passenv=PATH,GEM_HOME,GEM_PATH --loglevel=trace --port=4000 --devconsole ./chat.rb
#
# Then go to `http://localhost:4000` in two separate tabs
#
require 'redis'

# We need two redis connections, since the subscriber is blocking
$publisher, $subscriber = Redis.new, Redis.new

# Disable buffering
STDOUT.sync = true

# Input loop: Get a string from STDIN and publish to redis
def input
  puts "Choose a username:"
  $user = gets.strip
  loop do
    message = gets
    $publisher.publish :chat, "[#{$user}] #{message}" if message
  end
end

# Output loop: Subscribe to the redis chat channel and print all messages
# unless they start with my user name
def output
  $subscriber.subscribe :chat do |on|
    on.message do |channel, message|
      puts message unless message =~ /^\[#{$user}\]/
    end
  end
end

def run
  # Run output loop in a separate thread, and input loop
  Thread.new { output }
  input

rescue SystemExit, Interrupt
  # When a user leaves, publish this to the chat
  $publisher.publish :chat, "#{$user} has left the building"
  exit

end

run