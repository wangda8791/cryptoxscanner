BUILD :=	$(shell cat ../BUILD)

all: update-build-number
	./node_modules/.bin/ng build --prod --aot

update-build-number:
	npm run update-build $(BUILD)

clean:
	rm -rf dist

distclean: clean
	rm -rf node_modules

install-deps:
	npm install
