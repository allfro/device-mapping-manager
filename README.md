# device-volume-driver

This maps and enables devices into containers running on docker swarm. It is currently only compatible with linux systems that use cgroup v1. Contributions
for cgroupv2 are welcome.

# Installation

`docker plugin install redcanari/dvd`

# Usage

```yaml
version: "3.8"

volumes:
  dev_fuse:
    driver: redcanari/dvd
    driver_opts:
      device: /dev/fuse
  dev_dri_card0:
    driver: redcanari/dvd
    driver_opts:
      device: /dev/dri/card0
  dev_dri_renderD128:
    driver: redcanari/dvd
    driver_opts:
      device: /dev/dri/renderD128

services:
  rdesktop:
    image: lscr.io/linuxserver/rdesktop
    volumes:
      - dev_fuse:/dev/fuse
      - dev_dri_card0:/dev/dri/card0
      - dev_dri_renderD128:/dev/dri/renderD128
    ports:
      - 3389:3389

```
