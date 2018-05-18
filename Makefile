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
