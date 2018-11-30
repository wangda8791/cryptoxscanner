APP :=		cryptoxscanner
VERSION ?=	$(shell git rev-parse --abbrev-ref HEAD)

GOPATH ?=	${HOME}/go
CGO_ENABLED :=	1
TAGS :=		json1

LDFLAGS :=	-w -s

.PHONY:		dist $(APP)

all: build

build:
	./update-proto-version.py
	cd webapp && make
	$(GOPATH)/bin/packr -z
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

dist: GOOS=$(shell go env GOOS)
dist: GOARCH=$(shell go env GOARCH)
dist: GOEXE=$(shell go env GOEXE)
dist: OUTDIR=$(APP)-$(VERSION)$(VSUFFIX)-$(GOOS)-$(GOARCH)
dist: OUTBIN=$(APP)$(GOEXE)
dist:
	rm -rf dist/$(OUTDIR)
	mkdir -p dist/$(OUTDIR)
	cd webapp && $(MAKE)
	$(GOPATH)/bin/packr -z
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		go build --tags "$(TAGS)" --ldflags "$(LDFLAGS)" \
			-o dist/$(OUTDIR)/$(OUTBIN)
	(cd dist && zip -r $(OUTDIR).zip $(OUTDIR))
