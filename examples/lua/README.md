The examples demonstrate the use of websocketd with lua. There are two examples in the directory both very basic.

1. Greeter.lua simply echos back any input made from the client
2. json_ws.lua echos back any input from the client *after* converting it into a json string

It is pretty simple to extend these examples into full fledged applications. All you need is an stdin input loop

```
local input = io.stdin:read()

while input do
-- do anything here

-- update the input
input = io.stdin:read()

end


```

any thing you `print` goes out to the websocket client

Libraries and third party modules can be used by the standard `require` statement in lua.

## Running the examples



##### 1. Download

[Install](https://github.com/joewalnes/websocketd/wiki/Download-and-install) websocketd and add it to your `PATH`.

##### 2. Start a server: greeter

Run `websocketd --port=8080 --devconsole lua ./greeter.lua` and then go to `http://localhost:8080` to interact with it

##### 3. Start a server: json_ws

Run `websocketd --port=8080 --devconsole  lua  ./json_ws.lua` and then go to `http://localhost:8080` to interact with it

If you are using luajit instead of lua you may run the examples like this
(this assumes that you've got luajit in your path)

`websocketd --port=8080 --devconsole luajit ./json_ws.lua` and then go to `http://localhost:8080`