clean:
	rm -rf _out/

macos:
	rm -rf _out/macos
	mkdir -p _out/macos

	cargo build \
		--bin vencord-installer-gui \
		--bin vencord-installer-cli \
		--release \
		--target x86_64-apple-darwin \
		--target aarch64-apple-darwin

	cp -R package/macos/Vencord\ Installer.app _out/macos/

	lipo -create \
		target/x86_64-apple-darwin/release/vencord-installer-gui \
		target/aarch64-apple-darwin/release/vencord-installer-gui \
		-output _out/macos/Vencord\ Installer.app/Contents/MacOS/Vencord\ Installer

	@VERSION=$$(awk '/\[workspace.package\]/,/^$$/' Cargo.toml | sed -nE 's/version *= *"([^"]*)".*/\1/p'); \
		/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $$VERSION" _out/macos/Vencord\ Installer.app/Contents/Info.plist; \
		/usr/libexec/PlistBuddy -c "Set :CFBundleVersion $$VERSION" _out/macos/Vencord\ Installer.app/Contents/Info.plist

	lipo -create \
		target/x86_64-apple-darwin/release/vencord-installer-cli \
		target/aarch64-apple-darwin/release/vencord-installer-cli \
		-output _out/macos/vencord-installer-cli

windows:
	rm -rf _out/windows
	mkdir -p _out/windows

	cargo build \
		--bin vencord-installer-gui \
		--bin vencord-installer-cli \
		--release \
		--target x86_64-pc-windows-gnu

	cp target/x86_64-pc-windows-gnu/release/vencord-installer-gui.exe _out/windows/Vencord\ Installer.exe
	cp target/x86_64-pc-windows-gnu/release/vencord-installer-cli.exe _out/windows/vencord-installer-cli.exe

linux:
	rm -rf _out/linux
	mkdir -p _out/linux

	cargo build \
		--bin vencord-installer-gui \
		--bin vencord-installer-cli \
		--release \
		--target x86_64-unknown-linux-gnu

	cp target/x86_64-unknown-linux-gnu/release/vencord-installer-gui _out/linux/vencord-installer-gui
	cp target/x86_64-unknown-linux-gnu/release/vencord-installer-cli _out/linux/vencord-installer-cli
