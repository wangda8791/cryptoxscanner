#! /bin/bash

groupmod --gid "${REAL_GID}" builder > /dev/null
usermod --uid "${REAL_UID}" builder > /dev/null
usermod --gid "${REAL_GID}" builder > /dev/null

chown -R builder.builder /home/builder/go
chown -R builder.builder /src/webapp/node_modules

exec su - builder bash -c "cd /src && PATH=/usr/local/go/bin:$PATH make install-deps build"


