.PHONY: all cli gui native run test native-test web site install-gui uninstall-gui package-deb package-appimage android-debug

BIN_DIR ?= $(HOME)/bin
DATA_DIR ?= $(if $(XDG_DATA_HOME),$(XDG_DATA_HOME),$(HOME)/.local/share)

KRYON_ARCH ?= $(shell uname -m)
KRYON_BUILD_DIR := vendor/kryon/build/linux-$(KRYON_ARCH)
KRYON_RUNTIME_SOURCES := $(shell find vendor/kryon/src vendor/kryon/include -type f)

all: cli gui

cli:
	go build -o gopass ./cmd/gopass

$(KRYON_BUILD_DIR)/libkryon.a: $(KRYON_RUNTIME_SOURCES) vendor/kryon/Makefile
	$(MAKE) -C vendor/kryon -f Makefile all

gui: $(KRYON_BUILD_DIR)/libkryon.a
	cd gui && go build -o ../build/gopass-gui .

native: all

run: gui
	./build/gopass-gui

install-gui: gui
	mkdir -p $(BIN_DIR) $(DATA_DIR)/applications $(DATA_DIR)/icons/hicolor/512x512/apps
	cp build/gopass-gui $(BIN_DIR)/gopass-gui
	cp packaging/linux/xyz.waozi.pass.desktop $(DATA_DIR)/applications/xyz.waozi.pass.desktop
	cp assets/app/icon.png $(DATA_DIR)/icons/hicolor/512x512/apps/gopass.png

uninstall-gui:
	rm -f $(BIN_DIR)/gopass-gui $(DATA_DIR)/applications/xyz.waozi.pass.desktop $(DATA_DIR)/icons/hicolor/512x512/apps/gopass.png

test:
	go test ./...
	cd gui && go test ./...

# Checks the C generator (used by the Android app) against the same fixed
# vectors as the Go tests.
native-test:
	cc -Wall -Wextra -O2 -Inative native/gopass_core.c native/gopass_core_test.c -o build/gopass_core_test
	./build/gopass_core_test

ANDROID_DIR := droid
ANDROID_ABIS ?= armeabi-v7a,arm64-v8a

android-debug:
	cd $(ANDROID_DIR) && ./gradlew assembleDebug -Pabi=$(ANDROID_ABIS)
	mkdir -p build
	cp $(ANDROID_DIR)/app/build/outputs/apk/debug/app-debug.apk build/gopass-android-debug.apk

web:
	./web/build.sh

site: web
	test -f build/site/index.html
	test -f build/site/app/gopass.wasm

package-deb: gui
	./scripts/package-deb.sh

package-appimage: gui
	./scripts/package-appimage.sh
