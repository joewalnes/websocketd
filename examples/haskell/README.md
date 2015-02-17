## Haskell examples

### Count

Start the server with

```
$ websocketd --port=8080 --devconsole --passenv PATH ./count.hs
```

The passing of `PATH` was required for me because a typical Haskell installation of `runhaskell` does not go into `/usr/bin` but more like `/usr/local/bin`.

### Greeter

The greeter server waits for a line of text to be sent, then sends back a greeting in response, and continues to wait for more lines to come.

```
$ websocketd --port=8080 --devconsole --passenv PATH ./greeter.hs
```
