#!/usr/bin/env bash

arch=`uname -m`
echo "$arch"
GIT_VERSION=`git symbolic-ref --short HEAD`
echo "$GIT_VERSION"
if [ "$arch"=="x86_64" ]
then
    echo "exec on x86_64"
    docker pull wurstmeister/kafka:2.12-2.3.0
	  docker tag wurstmeister/kafka:2.12-2.3.0 rongzer/blockchain-kafka:${GIT_VERSION}
else
    echo "exec on arm"
    docker build -f Dockerfile -t rongzer/blockchain-kafka:${GIT_VERSION} .
fi
