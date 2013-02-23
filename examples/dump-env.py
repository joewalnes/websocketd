#!/usr/bin/python
# Copyright 2013 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.


import os
import sys

# Standard CGI(ish) environment variables, as defined in
# http://tools.ietf.org/html/rfc3875
var_names = [
  'REMOTE_ADDR',
  'REMOTE_HOST',
  'REMOTE_PORT',
  'SERVER_NAME',
  'SERVER_PORT',
  'SERVER_PROTOCOL',
  'SERVER_SOFTWARE',
  'GATEWAY_INTERFACE',
  'REQUEST_METHOD',
  'SCRIPT_NAME',
  'PATH_INFO',
  'PATH_TRANSLATED',
  'QUERY_STRING',
  'UNIQUE_ID',
  'REQUEST_URI',
]
for var_name in var_names:
  print '%s=%s' % (var_name, os.environ.get(var_name, '<unset>'))
  sys.stdout.flush() # Remember to flush

# Additional HTTP headers
for var_name in os.environ:
  if var_name.startswith('HTTP_'):
    print '%s=%s' % (var_name, os.environ[var_name])
    sys.stdout.flush() # Remember to flush
