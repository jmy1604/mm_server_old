set -x

sh ./kill_all_server.sh

sleep 1s

nohup `pwd`/center_server &
sleep 1s
nohup `pwd`/rpc_server &
sleep 1s
nohup `pwd`/hall_server &
sleep 1s 
nohup `pwd`/hall_server -f `pwd`/../conf/hall_server2.cfg &
sleep 1s
nohup `pwd`/login_server &