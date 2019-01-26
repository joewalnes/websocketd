#!/usr/bin/hhvm
<?hh // strict

use namespace HH\Lib\Str;

<<__EntryPoint>>
async function dumpEnv(): Awaitable<void> {
  // Standard CGI(ish) environment variables, as defined in
  // http://tools.ietf.org/html/rfc3875
  $names = keyset[
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

	/* HH_IGNORE_ERROR[2050] using global variable */
  $server = dict($_SERVER);

  foreach($names as $name) {
    print Str\format("%s = %s\n", $name, idx($server, $name, '<unset>'));
  }
  
  // Additional HTTP headers
  foreach($server as $k => $v) {
     if ($k is string && Str\starts_with($k, 'HTTP_')) {
        print Str\format("%s = %s\n", $k, $v as string);
     }
  }
}
