.PHONY: all clean build help

PLATFORM 	?= $(shell go env GOOS)
ARCH		?= $(shell go env GOARCH)

VERSION 	?= $(shell git describe --tags --abbrev=0)
HASH 		:= $(shell git rev-parse --short HEAD | xargs)

# "Developer ID Application: <name> (<team-id>)" or "-"
IDENTITY 	?= -

GUI 		?= 0
DMG 		?= 0
WAYLAND 	?= 0

LDFLAGS 	?= -s -w -X 'vencordinstaller/buildinfo.InstallerGitHash=$(HASH)' -X 'vencordinstaller/buildinfo.InstallerTag=$(VERSION)'
TAGS    	?= static
CGO 		:= 0
POSTFIX 	:=

ifeq ($(PLATFORM),windows)
ifeq ($(GUI),0)
LDFLAGS 	+= -extldflags=-static
else
LDFLAGS 	+= -H=windowsgui -extldflags=-static
endif # GUI
endif # PLATFORM

ifeq ($(GUI),0)
TAGS 		+= cli
POSTFIX 	:= Cli
else
CGO 		:= 1
ifeq ($(WAYLAND),1)
TAGS 		+= wayland
endif # WAYLAND
endif # GUI

# macOS specific
MACOS_ARCHS := $(ARCH)
ifeq ($(ARCH),universal)
MACOS_ARCHS := amd64 arm64
endif

all: build

build:
	mkdir -p build

ifeq ($(PLATFORM),darwin)
	for arch in $(MACOS_ARCHS); do \
		MACOSX_DEPLOYMENT_TARGET=10.13 \
		CGO_ENABLED=$(CGO) \
		GOOS=$(PLATFORM) \
		GOARCH=$$arch \
		go build \
			-v \
			-tags "$(TAGS)" \
			-ldflags "$(LDFLAGS)" \
			-o build/installer-$$arch; \
	done
ifeq ($(ARCH),universal)
	lipo -create \
		build/installer-amd64 \
		build/installer-arm64 \
		-output build/installer-universal
	rm build/installer-amd64 build/installer-arm64
endif # ARCH
ifeq ($(GUI),1)
	cp -R macos/VencordInstaller.app build/VencordInstaller.app
	/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $(VERSION)" build/VencordInstaller.app/Contents/Info.plist
	/usr/libexec/PlistBuddy -c "Set :CFBundleVersion $(HASH)" build/VencordInstaller.app/Contents/Info.plist
ifeq ($(ARCH),universal)
	mv build/installer-universal build/VencordInstaller.app/Contents/MacOS/VencordInstaller
else
	mv build/installer-$(ARCH) build/VencordInstaller.app/Contents/MacOS/VencordInstaller
endif # ARCH
	codesign --deep --force --options runtime --sign "$(IDENTITY)" build/VencordInstaller.app
ifeq ($(DMG),1)
ifeq ($(shell command -v create-dmg 2>/dev/null),)
	$(error create-dmg is not installed)
endif
	create-dmg \
		--volname "Vencord Installer" \
		--volicon macos/dmg.icns \
		--background "macos/background.png" \
		--window-pos 200 120 \
		--window-size 510 340 \
		--icon-size 100 \
		--icon VencordInstaller.app 160 155 \
		--hide-extension VencordInstaller.app \
		--app-drop-link 350 155 \
		build/VencordInstaller.dmg \
		build/VencordInstaller.app
endif # DMG
else # GUI
	mv build/installer-$(ARCH) build/VencordInstallerCli-$(PLATFORM)
endif # GUI

else ifeq ($(PLATFORM),windows) # PLATFORM
	export GOROOT=/mingw64/lib/go
	export GOPATH=/mingw64

	go-winres make --product-version "git-tag"
	CGO_ENABLED=$(CGO) GOOS=$(PLATFORM) GOARCH=$(ARCH) \
		go build -v -tags "$(TAGS)" \
		-ldflags "$(LDFLAGS)" \
		-o build/VencordInstaller$(POSTFIX).exe

else # PLATFORM
	CGO_ENABLED=$(CGO) GOOS=$(PLATFORM) GOARCH=$(ARCH) go build \
		-v \
		-tags "$(TAGS)" \
		-ldflags "$(LDFLAGS)" \
		-o build/VencordInstaller$(POSTFIX)-$(PLATFORM)
	chmod +x build/VencordInstaller$(POSTFIX)-$(PLATFORM)
endif

clean:
	rm -rf build

help:
	@printf '%s\n' \
		'Usage: make <target> [OPTIONS]' \
		'' \
		'Targets:' \
		'  help     Show this help page' \
		'  build    Build the project (default)' \
		'  clean    Remove build artifacts' \
		'' \
		'Options:' \
		'  GUI=1       Build with GUI support' \
		'  DMG=1       Build DMG for macOS' \
		'  WAYLAND=1        Build with Wayland support for Linux' \
		'  PLATFORM=<os>    Target platform (darwin, windows, linux)' \
		'  ARCH=<arch>      Target architecture (amd64, arm64, 386, universal (macOS only))' \
		'  IDENTITY=<id>    Code signing identity for macOS (default: -)' \
		'  VERSION=<ver>    Version string (default: latest git tag)' \
		'' \
		'Examples:' \
		'  make clean' \
		'  make PLATFORM=universal GUI=1' \
		'  make PLATFORM=linux ARCH=amd64' \
		'  make GUI=1 VERSION="1.0.0"'