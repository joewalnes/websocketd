## Haskell examples

### Count

Start the server with

```
$ websocketd --port=8080 --devconsole --passenv PATH ./count.hs
```

The passing of `PATH` was required for me because a typical Haskell installation of `runhaskell` does not go into `/usr/bin` but more like `/usr/local/bin`.
