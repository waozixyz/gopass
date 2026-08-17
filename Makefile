.PHONY: all cli gui native run test web site

KRYON_ARCH ?= $(shell uname -m)
KRYON_BUILD_DIR := vendor/kryon/build/linux-$(KRYON_ARCH)

all: cli gui

cli:
	go build -o build/gopass ./cmd/gopass

$(KRYON_BUILD_DIR)/libkryon.a:
	$(MAKE) -C vendor/kryon -f Makefile all

gui: $(KRYON_BUILD_DIR)/libkryon.a
	cd gui && go build -o ../build/gopass-gui .

native: all

run: gui
	./build/gopass-gui

test:
	go test ./...
	cd gui && go test ./...

web:
	./web/build.sh

site: web
	test -f build/site/index.html
	test -f build/site/app/gopass.wasm
