' Copyright 2013 Joe Walnes and the websocketd team.
' All rights reserved.
' Use of this source code is governed by a BSD-style
' license that can be found in the LICENSE file.


' Standard CGI(ish) environment variables, as defined in
' http://tools.ietf.org/html/rfc3875
names = Array(_
  "AUTH_TYPE", _
  "CONTENT_LENGTH", _
  "CONTENT_TYPE", _
  "GATEWAY_INTERFACE", _
  "PATH_INFO", _
  "PATH_TRANSLATED", _
  "QUERY_STRING", _
  "REMOTE_ADDR", _
  "REMOTE_HOST", _
  "REMOTE_IDENT", _
  "REMOTE_PORT", _
  "REMOTE_USER", _
  "REQUEST_METHOD", _
  "REQUEST_URI", _
  "SCRIPT_NAME", _
  "SERVER_NAME", _
  "SERVER_PORT", _
  "SERVER_PROTOCOL", _
  "SERVER_SOFTWARE", _
  "UNIQUE_ID", _
  "HTTPS"_
)

set shell = WScript.CreateObject("WScript.Shell")
set env = shell.Environment("PROCESS")

for each name in names
  value = env(name)
  if value = "" then
    value = "<unset>"
  end if
  WScript.echo name & "=" & value
next

for each item in env
  if instr(1, item, "HTTP_", 1) = 1 then
    WScript.Echo item
  end if
next
