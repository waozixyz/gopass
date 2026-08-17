.PHONY: all cli gui native run test web site install-gui uninstall-gui package-deb package-appimage

BIN_DIR ?= $(HOME)/bin
DATA_DIR ?= $(if $(XDG_DATA_HOME),$(XDG_DATA_HOME),$(HOME)/.local/share)

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

install-gui: gui
	mkdir -p $(BIN_DIR) $(DATA_DIR)/applications $(DATA_DIR)/icons/hicolor/512x512/apps
	cp build/gopass-gui $(BIN_DIR)/gopass-gui
	cp packaging/linux/xyz.waozi.gopass.desktop $(DATA_DIR)/applications/xyz.waozi.gopass.desktop
	cp assets/app/icon.png $(DATA_DIR)/icons/hicolor/512x512/apps/gopass.png

uninstall-gui:
	rm -f $(BIN_DIR)/gopass-gui $(DATA_DIR)/applications/xyz.waozi.gopass.desktop $(DATA_DIR)/icons/hicolor/512x512/apps/gopass.png

test:
	go test ./...
	cd gui && go test ./...

web:
	./web/build.sh

site: web
	test -f build/site/index.html
	test -f build/site/app/gopass.wasm

package-deb: gui
	./scripts/package-deb.sh

package-appimage: gui
	./scripts/package-appimage.sh
