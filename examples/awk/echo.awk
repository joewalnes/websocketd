#!/usr/bin/awk -f
# usage: 
#    websocketd --devconsole --port=7777 ./echo.awk
#
# then surf to http://localhost:7777 press the connect-button
# and type "echo hello" (enter)

function echo(argv,b,c,d,e){
	print "RECVD: "argv
}

{
	f = $1
	@f($0,$2,$3,$4,$5)
	fflush()
}
