#!/bin/sh

arch=`uname -m`
echo "$arch"
if [ "$arch"=="x86_64" ]
then
    echo "exec on x86_64"
    docker build -f Dockerfile -t rongzer/blockchain-buildenv:1.13.3 .
else
    echo "exec on arm"
    docker build -f arm-dockerfile/Dockerfile -t rongzer/blockchain-buildenv:1.13.3 .
fi

