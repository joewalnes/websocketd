#!/bin/bash
socketstate=`ps -efw | grep websocketd | grep 8080 | wc -l`

if [ ${socketstate} -gt 0 ]; then 
echo -e "Websocketd on port 8080 is already running"
echo -e "please kill manually\n"
ps -efw | grep websocketd
exit
fi

./websocketd --port=8080 --devconsole ./command.sh
echo -e "Quitting command_starter\n"
