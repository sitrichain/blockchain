#!/usr/bin/env bash

#nohup nsqlookupd >> /usr/local/output.log 2>&1 < /dev/null &
#nohup nsqd --lookupd-tcp-address=127.0.0.1:4160 >> /usr/local/output.log 2>&1 < /dev/null &
#nohup nsqadmin --lookupd-http-address=127.0.0.1:4161 >> /usr/local/output.log 2>&1 < /dev/null &
#
#exec peer node start --peer-defaultchain=false