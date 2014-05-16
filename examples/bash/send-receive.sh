#!/bin/bash


while true; do
	cnt=0
	while read -t 0.01 line; do
		cnt=$(($cnt + 1))
	done

	echo `date` "($cnt line(s) received)"
	sleep $((RANDOM % 10 + 1)) & wait
done
