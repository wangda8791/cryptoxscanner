all: build

build:
	./update-proto-version.py
	cd webapp && make
	go build

clean:
	rm -f cryptoxscanner
	cd webapp && $(MAKE) $@
	find . -name \*~ -delete
