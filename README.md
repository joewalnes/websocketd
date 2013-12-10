websocketd
==========

`websocketd` is a small command line tool that will wrap an existing command line interface program, and allow it to be accessed
via a WebSocket. It provides a quick mechanism for allowing web-applications to interact with existing command line tools.

WebSocket servers can now be built very easily. So long as you can write an executable program that reads `STDIN` and writes `STDOUT`, you
can build a WebSocket server. Do it in Python, Ruby, Perl, Bash, .NET, C, Go, PHP, Java, Clojure, Scala, Groovy, Expect, Awk, VBScript,
Haskell, Lua, Rwhatever! No networking libraries necessary.


Details
-------

Upon startup, `websocketd` will start a WebSocket server on the specified port, and listen for connections.

Upon a connection, it will fork the appropriate process, and disconnect the process when the WebSocket connection closes (and vice-versa).

Any message sent from the WebSocket client will be piped to the process `STDIN` stream, followed by a `\n` newline.

Any text printed by the process to `STDOUT` shall be sent as a WebSocket message whenever a `\n` newline is encountered.


Download
--------

**[Download for Linux, OS X and Windows](https://github.com/joewalnes/websocketd/wiki/Download-and-install)**


Quickstart
----------

To get started, we'll create a WebSocket endpoint that will accept connections, then send back
messages, counting to 10 with 1 second pause between each one, before disconnecting.

To show how simple it is, let's do it in Bash!

__count.sh__:

```sh
#!/bin/bash
for COUNT in $(seq 1 10); do
  echo $COUNT
  sleep 1
done
```

Before turning it into a WebSocket server, let's test it from the command line. The beauty of `websocketd` is servers
work equally well on the command line, or in shell scripts, as they do in the server - with no modifications required.

```sh
$ chmod +x count.sh
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
$ websocketd --port=8080 ./count.sh
```

Finally, let's create a web-page that to test it.

__count.html__:

```html
<!DOCTYPE html>
<pre id="log"></pre>
<script>
  // helper function: log message to screen
  function log(msg) {
    document.getElementById('log').innerText += msg + '\n';
  }

  // setup websocket with callbacks
  var ws = new WebSocket('ws://localhost:8080/');
  ws.onopen = function() {
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
Open this page in your web-browser. It will even work if you open it directly
from disk using a `file://` URL.


User Manual
-----------

**[More documentation in the user manual](https://github.com/joewalnes/websocketd/wiki)**

Example Project
---------------

*   [Plot real time Linux CPU/IO/Mem stats to a dashboard using websocketd and vmstat](https://github.com/joewalnes/web-vmstats)

Got more examples? Open a pull request.

=======
