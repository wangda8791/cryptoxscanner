#! /bin/sh

# Do we have a tty?
docker_it=""
if [ -t 1 ] ; then
    docker_it="-it"
fi

docker build -t cryptoxscanner-builder -f build/Dockerfile .
mkdir -p .docker_cache
docker run --rm ${docker_it} \
       -v `pwd`:/src \
       -v `pwd`/.docker_cache/node_modules:/src/webapp/node_modules \
       -v `pwd`/.docker_cache/go:/home/builder/go \
       -w /src \
       -e REAL_UID=`id -u` -e REAL_GID=`id -g` \
       cryptoxscanner-builder "make install-deps dist"
