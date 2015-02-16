import redis

let redisClient = open()

while true:
  let id = readLine(stdin)
  if id != "":
    let votes = redisClient.incr("voter/" & id)
    echo votes
    flushFile stdout
