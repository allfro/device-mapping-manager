# device-volume-driver

This maps and enables devices into containers running on docker swarm. It is currently only compatible with linux systems that use cgroup v1. Contributions
for cgroupv2 are welcome.

# Installation

`docker stack deploy -c docker-compose.yml dmm`

# Usage

```yaml
version: "3.8"

services:
  rdesktop:
    image: lscr.io/linuxserver/rdesktop
    volumes:
      - /dev/dri:/dev/dri
    ports:
      - 3389:3389

```
