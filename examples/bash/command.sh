#!/bin/bash
mypid=$$                                  # remember the pid of this script for later cleanup
wspid=$(ps -efw | grep websocketd | grep 8080 | awk '{printf $2}')  # get PID of websocket process

ctrl_c() {                                # define action on Ctrl-C or INT Signal
        echo "Trapped CTRL-C" > /dev/tty
        rm -f $pipefile                   # rm fifos
        exit
}
trap ctrl_c INT                           # trap INT

if [ -z "$wspid" ]; then echo "No websocket running"; exit; fi
echo "The wspid is $wspid" > /dev/tty
export DISPLAY=:0                         # set display to localhost, this is needed to start commands which open a GUI on the X11 screen
echo "Connected to bash"                  # write welcome message to html element in the web browser 


#setup pipe or socat
pipefile="/tmp/${mypid}_fifo"            # we need a pipe for every command instance
echo "Pipe:$pipefile" > /dev/tty

#pure pipe
rm  -f $pipefile                         # delete and make fifo
mkfifo $pipefile 
bash < $pipefile 2>&1 &                  # execute commands in backround

#socat pipe, removed ctty from options (no output from login process in terminal with this option)
#/usr/bin/socat -u OPEN:$pipefile EXEC:'/bin/bash',pty,setsid,stderr &
exec 3> $pipefile                        # output descriptor 3 to pipe  
 

while [ 1 ]
  do
    while read -t 1 line; do              # read line by line from stdin with timeout 
    echo $line                            # write to html element
    #echo "$line" > $pipefile             # to socat pipe
    echo "$line" >&3                      # to normal pipe
    echo "command: $line" > /dev/tty      # to shell where script is started
  done < /dev/stdin
  #echo !State: connected $(date)          # send state and current date,time to html page in browser 
  if [ ! -e /proc/$wspid ] ; then         # cleanup and quit script if websocket is not available
    echo -e "Websocket not available, quitting"
    exec 3>&-                             #close pipe
    rm /tmp/$pipefile
    kill -9 $mypid
    exit
  fi
done



