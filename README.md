websocketd
==========

`WebSocket` is a small command-line tool that will wrap an existing command-line interface program, and allow it to be accessed via a WebSocket.

WebSocket-capable applications can now be built very easily. As long as you can write an executable program that reads `STDIN` and writes to `STDOUT`, you can build a WebSocket server. Do it in Python, Ruby, Perl, Bash, .NET, C, Go, PHP, Java, Clojure, Scala, Groovy, Expect, Awk, VBScript, Haskell, Lua, R, whatever! No networking libraries are necessary.

-[@joewalnes](https://twitter.com/joewalnes)

Details
-------

Upon startup, `WebSocket` will start a WebSocket server on a specified port, and listen for connections.

Upon a connection, it will fork the appropriate process, and disconnect the process when the WebSocket connection closes (and vice-versa).

Any message sent from the WebSocket client will be piped to the process's `STDIN` stream, followed by a `\n` newline.

Any text printed by the process to `STDOUT` shall be sent as a WebSocket message whenever a `\n` newline is encountered.


Download
--------

If you're on a Mac, you can install `WebSocket` using [Homebrew](http://brew.sh/). Just run `brew install WebSocket`. For other operating systems, or if you don't want to use Homebrew, check out the link below.

**[Download for Linux, OS X and Windows](https://github.com/joewalnes/websocketd/wiki/Download-and-install)**


Quickstart
----------

To get started, we'll create a WebSocket endpoint that will accept connections, and then send back messages, counting to 10 with 1 second pause between each one, before disconnecting.

To show how simple it is, let's do it in Bash!

__count.sh__:

```sh
#!/bin/bash
for ((COUNT = 1; COUNT <= 10; COUNT++)); do
  echo $COUNT
  sleep 1
done
```

Before turning it into a WebSocket server, let's test it from the command line. The beauty of `WebSocket` is that servers work equally well in the command line, or in shell scripts, as they do in the server - with no modifications required.

```sh
$ chmod +x count. sh
$ ./count.sh
1
2
3
4
5
6
7
8
9
10
```

Now let's turn it into a WebSocket server:

```sh
$ web sockets --port=8080 ./count.sh
```

Finally, let's create a web page to test it.

__count.html__:

```HTML
<!DOCTYPE html>
<pre id="log"></pre>
<script>
  // helper function: log message to screen
  function log(msg) {
    document.getElementById('log').textContent += msg + '\n';
  }

  // setup web socket with callbacks
  var ws = new WebSocket('ws://localhost:8080/');
  ws.on open = function() {
    log('CONNECT');
  };
  ws.onclose = function() {
    log('DISCONNECT');
  };
  ws.onmessage = function(event) {
    log('MESSAGE: ' + event.data);
  };
</script>
```
Open this page in your web browser. It will even work if you open it directly
from disk using a `file://` URL.

More Features
-------------

*   Very simple install. Just [download](https://github.com/joewalnes/websocketd/wiki/Download-and-install) the single executable for Linux, Mac or Windows and run it. Minimal dependencies, no installers, no package managers, and no external libraries. Suitable for development and production servers.
*   Server-side scripts can access details about the WebSocket HTTP request (e.g. remote host, query parameters, cookies, path, etc) via standard [CGI environment variables](https://github.com/joewalnes/websocketd/wiki/Environment-variables).
*   As well as serving websocket daemons it also includes a static file server and classic CGI server for convenience.
*   Command line help available via `WebSocket --help`.
*   Includes [WebSocket developer console](https://github.com/joewalnes/websocketd/wiki/Developer-console) to make it easy to test your scripts before you've built a JavaScript frontend.
*   [Examples in many programming languages](https://github.com/joewalnes/websocketd/tree/master/examples) are available to help you getting started.

User Manual
-----------

**[More documentation in the user manual](https://github.com/joewalnes/websocketd/wiki)**

Example Projects
----------------

*   [Plot real-time Linux CPU/IO/Mem stats to an HTML5 dashboard using WebSockets and vmstat](https://github.com/joewalnes/web-vmstats) _(for Linux)_
*   [Arbitrary REPL in the browser using websocketd](https://github.com/rowanthorpe/ws-repl)
*   [Retrieve SQL data from server with LiveCode and webSocketd](https://github.com/samansjukur/wslc)
*   [List files from a configured folder](https://github.com/dbalakirev/directator) _(for Linux)_
*   [Listen for gamepad events and report them to the system](https://github.com/experiment322/controlloid-server) _(this + android = gamepad emulator)_

Got more examples? Open a pull request.

My Other Projects
-----------------

*   [ReconnectingWebSocket](https://github.com/joewalnes/reconnecting-websocket) - The simplest way to add some robustness to your WebSocket connections.
*   [Smoothie Charts](http://smoothiecharts.org/) - JavaScript charts for streaming data.
*   Visit [The Igloo Lab](http://theigloolab.com/) to see and subscribe to other things I make.

And [follow @joewalnes](https://twitter.com/joewalnes)!
