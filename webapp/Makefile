BUILD :=	$(shell cat ../BUILD)

all: update-build
	./node_modules/.bin/ng build --prod --aot

update-build:
	npm run update-build $(BUILD)

clean:
	rm -rf dist

distclean: clean
	rm -rf node_modules

install-deps:
	npm install
