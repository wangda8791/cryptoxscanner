#! /bin/bash

sudo groupmod --gid "${REAL_GID}" builder > /dev/null
sudo usermod --uid "${REAL_UID}" builder > /dev/null
sudo usermod --gid "${REAL_GID}" builder > /dev/null

sudo chown -R builder.builder /home/builder/go
sudo chown -R builder.builder /src/webapp/node_modules

#exec su - builder bash -c "cd /src && PATH=/usr/local/go/bin:$PATH make install-deps build"

make install-deps build

