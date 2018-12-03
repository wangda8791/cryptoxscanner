APP :=		cryptoxscanner
VERSION ?=	$(shell git rev-parse --abbrev-ref HEAD)

GOPATH ?=	${HOME}/go
CGO_ENABLED :=	1
TAGS :=		json1

BUILD :=	$(shell cat ./BUILD)

BUILD_GO_VAR :=	gitlab.com/crankykernel/cryptoxscanner/pkg.BuildNumber

LDFLAGS :=	-w -s \
		-X \"$(BUILD_GO_VAR)=$(BUILD)\"

.PHONY:		build dist $(APP)

all: build

build:
	cd webapp && make
	$(GOPATH)/bin/packr -z -v -i server
	$(MAKE) $(APP)

$(APP):
	go build --tags "$(TAGS)" -ldflags "$(LDFLAGS)"

install-deps:
	$(MAKE) -C webapp $@
	go get github.com/gobuffalo/packr/packr
	go mod download

clean:
	rm -f cryptoxscanner
	cd webapp && $(MAKE) $@
	find . -name \*~ -delete
	find . -name \*-packr.go -delete
	rm -rf dist

distclean:
	cd webapp && $(MAKE) $@
	rm -rf vendor

docker-build:
	docker build -t cryptoxscanner-builder -f build/Dockerfile.build .
	docker run --rm -it \
		-v `pwd`:/src \
		-v `pwd`/.cache/node_modules:/src/webapp/node_modules \
		-v `pwd`/.cache/go:/home/builder/go \
		-w /src \
		-e REAL_UID=`id -u` -e REAL_GID=`id -g` \
		cryptoxscanner-builder make install-deps build

dist: GOOS=$(shell go env GOOS)
dist: GOARCH=$(shell go env GOARCH)
dist: GOEXE=$(shell go env GOEXE)
dist: OUTDIR=$(APP)-$(VERSION)$(VSUFFIX)-$(GOOS)-$(GOARCH)
dist: OUTBIN=$(APP)$(GOEXE)
dist:
	rm -rf dist/$(OUTDIR)
	mkdir -p dist/$(OUTDIR)
	cd webapp && $(MAKE)
	$(GOPATH)/bin/packr -z -v -i server
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		go build --tags "$(TAGS)" --ldflags "$(LDFLAGS)" \
			-o dist/$(OUTDIR)/$(OUTBIN)
	(cd dist && zip -r $(OUTDIR).zip $(OUTDIR))
