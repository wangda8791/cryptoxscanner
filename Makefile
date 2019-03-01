APP :=		cryptoxscanner
ifndef VERSION
VERSION :=	0.1.0dev$(shell date +%s)
endif

.PHONY:		build dist

all: build

build:
	cd webapp && make
	cd go && make

update-build-number:
	(cd webapp && make update-build-number)

install-deps:
	cd webapp && $(MAKE) $@
	cd go && $(MAKE) $@

clean:
	cd webapp && $(MAKE) $@
	cd go && $(MAKE) $@
	find . -name \*~ -delete
	find . -name \*-packr.go -delete
	rm -rf dist

distclean: clean
	cd go && $(MAKE) $@
	cd webapp && $(MAKE) $@

docker-build:
	docker build -t cryptoxscanner-builder -f build/Dockerfile .
	mkdir -p .docker_cache
	docker run --rm -it \
		-v `pwd`:/src \
		-v `pwd`/.docker_cache/node_modules:/src/webapp/node_modules \
		-v `pwd`/.docker_cache/go:/home/builder/go \
		-w /src \
		-e REAL_UID=`id -u` -e REAL_GID=`id -g` \
		cryptoxscanner-builder make install-deps build

docker-dist:
	docker build -t cryptoxscanner-builder -f build/Dockerfile .
	mkdir -p .docker_cache
	docker run --rm -it \
		-v `pwd`:/src \
		-v `pwd`/.docker_cache/node_modules:/src/webapp/node_modules \
		-v `pwd`/.docker_cache/go:/home/builder/go \
		-w /src \
		-e REAL_UID=`id -u` -e REAL_GID=`id -g` \
		cryptoxscanner-builder "make install-deps dist"

dist: GOOS=$(shell go env GOOS)
dist: GOARCH=$(shell go env GOARCH)
dist: DIR = $(APP)-$(VERSION)-$(GOOS)-$(GOARCH)
dist:
	rm -rf dist/$(DIR) && mkdir -p dist/$(DIR)
	test "${SKIP_WEBAPP}" || (cd webapp && $(MAKE))
	cd go && $(MAKE) DIR=../dist/$(DIR)
	cp README.md LICENSE.txt dist/$(DIR)/
	cd dist && zip -r $(DIR).zip $(DIR)
