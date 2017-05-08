local input = io.stdin:read()
local json = require("json")
while input do
   print(json.encode({res="json",mess=input}))
   io.stdout:flush()
   input = io.stdin:read()
end
