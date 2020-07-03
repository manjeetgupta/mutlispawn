#!/bin/bash
LOGFILE=test.log

#rm -f $LOGFILE

echo "Launch.." >> $LOGFILE
./5spawn 103 &

echo "Start stage 1.." >> $LOGFILE
sleep 5
for i in {1..5}
do   
    echo "Round" $i >> $LOGFILE
    r=$(($RANDOM %10))
    if [ $((r % 2)) == 0 ]
    then
        pkill LOKI
    else
        pkill THANOS
    fi    
    sleep 5
done


echo "Start stage 2.." >> $LOGFILE
sleep 5
pkill 5spawn
pkill LOKI


echo "Start stage 3..Reparent" >> $LOGFILE
sleep 5
./5spawn 103 &

echo "Exit!" >> $LOGFILE
exit