#/bin/sh
nohup ./otgw2db-linux-amd64 &>> otgw2db.log &
echo $! > otgw2db_pid.txt
date >> otgw2db_pid.txt
ps a | grep otgw2db
