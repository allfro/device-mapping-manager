#!/bin/sh

set -eux

ROOTFS=plugin/rootfs
CONFIG=plugin/config.json

tag=redcanari/device-volume-driver
docker build -t "$tag" -f Dockerfile .
id=$(docker create "$tag" true)
rm -Rf $ROOTFS
mkdir -p $ROOTFS
docker export "$id" | tar -x -C $ROOTFS
docker rm -vf "$id"
docker rmi "$tag"
cp config.json $CONFIG

docker plugin rm -f redcanari/device-volume-driver || echo
docker plugin create redcanari/device-volume-driver ./plugin
docker plugin enable redcanari/device-volume-driver