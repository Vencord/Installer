.PHONY: all clean build

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
VERSION_RES ?= $(shell git describe --tags --always)
HASH 		:= $(shell git rev-parse --short HEAD)

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
	lipo -create _build/installer-amd64 _build/installer-arm64 -output _build/VencordInstaller
	rm _build/installer-amd64 _build/installer-arm64
ifeq ($(WITH_GUI),1)
	cp -R macos/VencordInstaller.app _build/VencordInstaller.app
	/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $(VERSION)" _build/VencordInstaller.app/Contents/Info.plist
	/usr/libexec/PlistBuddy -c "Set :CFBundleVersion $(HASH)" _build/VencordInstaller.app/Contents/Info.plist
	mv _build/VencordInstaller _build/VencordInstaller.app/Contents/MacOS/VencordInstaller
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
		--window-size 510 350 \
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