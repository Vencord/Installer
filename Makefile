.PHONY: all clean build help

PLATFORM 	?= $(shell go env GOOS)

ifeq ($(PLATFORM),darwin)
ARCHS 		?= amd64 arm64
IDENTITY 	?= -
else ifeq ($(PLATFORM),windows)
ARCHS 		?= amd64 #386
else
ARCHS 		?= amd64
endif

VERSION 	?= $(shell git describe --tags --abbrev=0)
HASH 		:= $(shell git rev-parse --short HEAD | xargs)
VERSION_RES := $(shell git describe --tags --always)

ifeq ($(PLATFORM),windows)
LDFLAGS 	:= -s -w -H=windowsgui -extldflags=-static -X 'vencordinstaller/buildinfo.InstallerGitHash=$(HASH)' -X 'vencordinstaller/buildinfo.InstallerTag=$(VERSION)'
else
LDFLAGS 	:= -s -w -X 'vencordinstaller/buildinfo.InstallerGitHash=$(HASH)' -X 'vencordinstaller/buildinfo.InstallerTag=$(VERSION)'
endif

WITH_GUI 	?= 0
WITH_DMG 	?= 0
WAYLAND 	?= 0
ifeq ($(WITH_GUI),0)
ifeq ($(WAYLAND),0)
TAGS 		:= "static cli"
else
TAGS 		:= "static cli wayland"
endif # WAYLAND
POSTFIX 	:= Cli
WITH_CGO 	?= 0
else
TAGS 		:= "static"
POSTFIX 	:=
WITH_CGO 	?= 1
endif

all: build

build:
	mkdir -p _build

ifeq ($(PLATFORM),darwin)
	for arch in $(ARCHS); do \
		MACOSX_DEPLOYMENT_TARGET=10.13 CGO_ENABLED=$(WITH_CGO) GOOS=$(PLATFORM) GOARCH=$$arch go build \
			-v \
			-tags $(TAGS) \
			-ldflags "$(LDFLAGS)" \
			-o _build/installer-$$arch; \
	done
ifeq ($(WITH_GUI),1)
	cp -R macos/VencordInstaller.app _build/VencordInstaller.app
	/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $(VERSION)" _build/VencordInstaller.app/Contents/Info.plist
	/usr/libexec/PlistBuddy -c "Set :CFBundleVersion $(HASH)" _build/VencordInstaller.app/Contents/Info.plist
	lipo -create _build/installer-amd64 _build/installer-arm64 -output _build/VencordInstaller.app/Contents/MacOS/VencordInstaller
	rm _build/installer-amd64 _build/installer-arm64
	codesign --deep --force --options runtime --sign "$(IDENTITY)" _build/VencordInstaller.app
ifeq ($(WITH_DMG),1)
ifeq ($(shell command -v create-dmg 2>/dev/null),)
	$(error create-dmg is not installed)
endif
	create-dmg \
		--volname "Vencord Installer" \
		--volicon macos/VencordInstaller.app/Contents/Resources/AppIcon.icns \
		--background "macos/background.png" \
		--window-pos 200 120 \
		--window-size 510 340 \
		--icon-size 100 \
		--icon VencordInstaller.app 160 155 \
		--hide-extension VencordInstaller.app \
		--app-drop-link 350 155 \
		_build/VencordInstaller.dmg \
		_build/VencordInstaller.app
endif # WITH_DMG
endif # WITH_GUI
else ifeq ($(PLATFORM),windows)
	export GOROOT=/mingw64/lib/go
	export GOPATH=/mingw64

	go-winres make --product-version $(VERSION_RES)
	CGO_ENABLED=$(WITH_CGO) GOOS=$(PLATFORM) GOARCH=$$arch \
		go build -v -tags $(TAGS) \
		-ldflags "$(LDFLAGS)" \
		-o _build/VencordInstaller$(POSTFIX).exe
else
	CGO_ENABLED=$(WITH_CGO) GOOS=$(PLATFORM) GOARCH=$(ARCHS) go build \
		-v \
		-tags $(TAGS) \
		-ldflags "$(LDFLAGS)" \
		-o _build/VencordInstaller$(POSTFIX)-$(PLATFORM)
	chmod +x _build/VencordInstaller$(POSTFIX)-$(PLATFORM)
endif

clean:
	rm -rf _build

help:
	@printf '%s\n' \
		'Usage: make <target> [OPTIONS]' \
		'' \
		'Targets:' \
		'  help        Show this help page' \
		'  build       Build the project (default)' \
		'  clean       Remove build artifacts' \
		'' \
		'Options:' \
		'  WITH_GUI=1           Build with GUI support' \
		'  WITH_DMG=1           Build DMG for macOS' \
		'  WAYLAND=1            Build with Wayland support for Linux' \
		'  PLATFORM=<os>        Target platform (darwin, windows, linux)' \
		'  ARCHS=<archs>        Target architectures (amd64, arm64, 386)' \
		'  IDENTITY=<id>        Code signing identity for macOS (default: -)' \
		'  VERSION=<ver>        Version string (default: latest git tag)' \
		'' \
		'Examples:' \
		'  make clean' \
		'  make WITH_GUI=1' \
		'  make PLATFORM=linux ARCHS=amd64' \
		'  make WITH_GUI=1 VERSION="1.0.0"'