all: update-version
	./node_modules/.bin/ng build --prod --aot

update-version:
	npm run update-build \
		$(shell cat ../VERSION.UI) $(shell cat ../VERSION.PROTO)
	cp ../VERSION.PROTO src/assets/VERSION.PROTO
	cp ../VERSION.UI src/assets/VERSION.UI

serve:
	./node_modules/.bin/ng serve

clean:
	rm -f src/assets/VERSION.PROTO src/assets/VERSION.UI
	rm -rf dist

distclean: clean
	rm -rf node_modules

install-deps:
	npm install
