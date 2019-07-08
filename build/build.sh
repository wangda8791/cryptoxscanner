#! /bin/sh

TAG="crankykernel/cryptoxscanner:builder"

# Do we have a tty?
docker_it=""
if [ -t 1 ] ; then
    docker_it="-it"
fi

prep() {
    docker build -t ${TAG} -f build/Dockerfile .
    mkdir -p .docker_cache
}

case "$1" in
    dist)
	prep
	docker run --rm ${docker_it} \
	       -v `pwd`:/src \
	       -w /src \
	       -e REAL_UID=`id -u` -e REAL_GID=`id -g` \
	       ${TAG} "make install-deps dist"
	;;

    *)
	prep
	docker run --rm ${docker_it} \
	       -v `pwd`:/src \
	       -w /src \
	       -e REAL_UID=`id -u` -e REAL_GID=`id -g` \
	       ${TAG} "make install-deps build"
	;;
esac
