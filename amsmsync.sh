#! /bin/sh

rm -f /root/Sync/*conflict*
if [[ -f /root/.config/AMSM/amsmp2p.conf ]]
then
s=$(grep HWID /root/.config/AMSM/amsmp2p.conf|cut -c 7-9)
s1=$((($(grep HWID /root/.config/AMSM/amsmp2p.conf|cut -c 7-9)%10)*20))
sed -i "s/SYNCTIME.*/SYNCTIME=\"$s1\"/" /root/.config/AMSM/amsmp2p.conf
fi
