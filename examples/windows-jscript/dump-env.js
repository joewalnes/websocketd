// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.


// Standard CGI(ish) environment variables, as defined in
// http://tools.ietf.org/html/rfc3875
var names = [
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
];

var shell = WScript.CreateObject("WScript.Shell");
var env = shell.Environment('PROCESS');

for (var i = 0; i < names.length; i++) {
  var name = names[i];
  var value = env(name) || '<unset>';
  WScript.echo(name + '=' + value);
}

for(var en = new Enumerator(env); !en.atEnd(); en.moveNext()) {
  var item = en.item();
  if (item.indexOf('HTTP_') == 0) {
    WScript.Echo(item);
  }
}