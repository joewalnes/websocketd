local input = io.stdin:read()
while input do
   print(input)
   io.stdout:flush()
   input = io.stdin:read()
end

