# Vencord Installer

Vencord Installer allows you to install [Vencord, a Discord Desktop client mod](https://github.com/Vendicated/Vencord)

This installer is still in testing, so use at your own Risk.

## Usage

### Windows

Download the latest VencordInstaller.exe from [GitHub Releases](https://github.com/Vendicated/VencordInstaller/releases/latest) and run it

### Linux

curl https://raw.githubusercontent.com/Vendicated/VencordInstaller/main/install.sh | sh

### Mac

Download the latest Mac build from [GitHub Releases](https://github.com/Vendicated/VencordInstaller/releases/latest), unzip it and run VencordInstaller.app

### Building from source

You need the go compiler and gcc (MinGW on Windows).

On Linux you also need some dependencies:

```sh
apt install -y pkg-config libsdl2-dev libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libglx-dev libgl1-mesa-dev libxxf86vm-dev
```

Then just run

```sh
go build
```

You might want to pass some flags to this command to get a better build.
See [the GitHub workflow(https://github.com/Vendicated/VencordInstaller/blob/main/.github/workflows/release.yml)] for what flags I pass or if you wan't more precise instructions
