#! /bin/bash

set -e
set -x

groupmod --gid "${REAL_GID}" builder
usermod --uid "${REAL_UID}" builder

# sudo groupmod --gid "${REAL_GID}" builder > /dev/null
# sudo usermod --uid "${REAL_UID}" builder > /dev/null
# sudo usermod --gid "${REAL_GID}" builder > /dev/null

chown -R builder.builder /home/builder/go
chown -R builder.builder /src/webapp/node_modules

#exec su - builder bash -c "cd /src && PATH=/usr/local/go/bin:$PATH make install-deps build"

su builder -m -c "HOME=/home/builder make install-deps build"


