#!/usr/bin/php
<?php

// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Standard CGI(ish) environment variables, as defined in
// http://tools.ietf.org/html/rfc3875
$names = array(
  'AUTH_TYPE',
  'CONTENT_LENGTH',
  'CONTENT_TYPE',
  'GATEWAY_INTERFACE',
  'PATH_INFO',
  'PATH_TRANSLATED',
  'QUERY_STRING',
  'REMOTE_ADDR',
  'REMOTE_HOST',
  'REMOTE_IDENT',
  'REMOTE_PORT',
  'REMOTE_USER',
  'REQUEST_METHOD',
  'REQUEST_URI',
  'SCRIPT_NAME',
  'SERVER_NAME',
  'SERVER_PORT',
  'SERVER_PROTOCOL',
  'SERVER_SOFTWARE',
  'UNIQUE_ID',
  'HTTPS'
);

foreach ($names as $name) {
  $value = isset($_SERVER[$name]) ? $_SERVER[$name] : '<unset>';
	echo $name . '=' . $value . "\n";
}

// Additional HTTP headers
foreach ($_SERVER as $name => $value) {
  if (strpos($name, 'HTTP_') === 0) {
    echo $name . '=' . $value . "\n";
  }
}
