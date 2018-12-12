APP :=		cryptoxscanner
VERSION ?=	$(shell git rev-parse --abbrev-ref HEAD)

GOPATH ?=	${HOME}/go
CGO_ENABLED :=	1
GO_TAGS :=	json1

BUILD :=	$(shell cat ../BUILD)

BUILD_GO_VAR :=	gitlab.com/crankykernel/cryptoxscanner/version.BuildNumber

GO_LDFLAGS :=	-w -s \
		-X \"$(BUILD_GO_VAR)=$(BUILD)\"

.PHONY:		$(APP)

all: $(APP)

$(APP): EXE := $(shell go env GOEXE)
$(APP): BIN := $(APP)$(EXE)
$(APP): DIR ?= .
$(APP):
	test -e ../webapp/dist && $(GOPATH)/bin/packr -z -v || true
	go build -o $(DIR)/$(BIN) --tags "$(GO_TAGS)" -ldflags "$(GO_LDFLAGS)"

install-deps:
	go get github.com/gobuffalo/packr/packr
	go mod download

clean:
	rm -f $(APP)
	find . -name \*~ -delete
	find . -name \*-packr.go -delete

distclean: clean
