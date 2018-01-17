1.run go build in `examples/go`
```bash
$ go build
```

2.run this commend in `examples`, open http://localhost:8080 with browser, and then connect to `ws://localhost:8080/go`

```bash
$ websocketd --port=8080 --dir=go --devconsole
```

or run this commend in `examples/go`, open `test.html` with browser

```bash
$ websocketd --port=8080 go
```