APP :=		cryptoxscanner
ifndef VERSION
VERSION :=	0.1.0dev$(shell date +%Y%m%d%H%M%S)
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
