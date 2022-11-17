#!/bin/sh

set -eux

ROOTFS=plugin/rootfs
CONFIG=plugin/config.json

tag=redcanari/dvd
docker build -t "$tag" -f Dockerfile .
id=$(docker create "$tag" true)
rm -Rf $ROOTFS
mkdir -p $ROOTFS
docker export "$id" | tar -x -C $ROOTFS
docker rm -vf "$id"
docker rmi "$tag"
cp config.json $CONFIG

docker plugin rm -f $tag || echo
docker plugin create $tag ./plugin
docker plugin push $tag
#docker plugin enable $tag