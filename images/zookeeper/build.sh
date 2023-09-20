#!/usr/bin/env bash

arch=`uname -m`
echo "$arch"
GIT_VERSION=`git symbolic-ref --short HEAD`
echo "$GIT_VERSION"
if [ "$arch"=="x86_64" ]
then
    echo "exec on x86_64"
    docker pull zookeeper:3.5
	docker tag zookeeper:3.5 rongzer/blockchain-zookeeper:${GIT_VERSION}
else
    echo "exec on arm"
    docker pull zookeeper:3.4.13
	docker tag zookeeper:3.4.13 rongzer/blockchain-zookeeper:${GIT_VERSION}
fi
