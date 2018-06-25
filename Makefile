APP :=		cryptoxscanner
VERSION ?=	$(shell git rev-parse --abbrev-ref HEAD)

# SQLite will be used soon.
CGO_ENABLED :=	1
TAGS :=		json1

.PHONY:		dist

all: build

build:
	./update-proto-version.py
	cd webapp && make
	packr -z
	go build -ldflags "-w -s"

install-deps:
	$(MAKE) -C webapp $@
	go get github.com/golang/dep/cmd/dep
	go get github.com/cespare/reflex
	go get github.com/gobuffalo/packr/packr
	dep ensure -v

clean:
	rm -f cryptoxscanner
	cd webapp && $(MAKE) $@
	find . -name \*~ -delete
	find . -name \*-packr.go -delete
	rm -rf dist

dist: GOOS=$(shell go env GOOS)
dist: GOARCH=$(shell go env GOARCH)
dist: GOEXE=$(shell go env GOEXE)
dist: OUTDIR=$(APP)-$(VERSION)$(VSUFFIX)-$(GOOS)-$(GOARCH)
dist: OUTBIN=$(APP)$(GOEXE)
dist:
	dep ensure
	rm -rf dist/$(OUTDIR)
	mkdir -p dist/$(OUTDIR)
	cd webapp && $(MAKE)
	packr -z
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		go build --tags "$(TAGS)" -o dist/$(OUTDIR)/$(OUTBIN)
	(cd dist && zip -r $(OUTDIR).zip $(OUTDIR))
