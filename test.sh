#!/bin/bash
LOGFILE=test.log
PAUSETIME=2

#rm -f $LOGFILE

echo "Launch.." >> $LOGFILE
./5spawn 103 &

# echo "Start stage 1.." >> $LOGFILE
# sleep $PAUSETIME
# for i in {1..5}
# do   
#     echo "Round" $i >> $LOGFILE
#     r=$(($RANDOM %10))
#     if [ $((r % 2)) == 0 ]
#     then
#         pkill LOKI
#     else
#         pkill THANOS
#     fi    
#     sleep $PAUSETIME
# done


echo "Start stage 2.." >> $LOGFILE
sleep $PAUSETIME
pkill 5spawn
pkill LOKI
pidthanos=$(pidof THANOS)
echo "Pid of app before reparent: "$pidthanos" " >> $LOGFILE

echo "Start stage 3..Reparent" >> $LOGFILE
sleep $PAUSETIME
./5spawn 103 &
pidthanos=$(pidof THANOS)
echo "Pid of app after reparent: "$pidthanos" " >> $LOGFILE

echo "Start stage 4..Kill reparented child" >>$LOGFILE
sleep $PAUSETIME
pidc=$(pidof THANOS)
kill -9 $pidc
echo "Pid of app to which SIGKILL is sent: "$pidthanos" " >> $LOGFILE
sleep 1
pidthanos=$(pidof THANOS)
echo "Pid of app after respawning: "$pidthanos" " >> $LOGFILE

sleep $PAUSETIME
echo "Exit!" >> $LOGFILE
exit
