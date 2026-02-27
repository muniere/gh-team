GO ?= go
BINARY ?= gh-team
PKG ?= .

.PHONY: build install clean bootstrap

bootstrap:
	$(GO) mod tidy

build:
	$(GO) build -o $(BINARY) $(PKG)

install: build
	gh extension install . --force

clean:
	rm -f $(BINARY)
