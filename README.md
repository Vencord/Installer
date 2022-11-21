# Vencord Installer

Vencord Installer allows you to install [Vencord, a Discord Desktop client mod](https://github.com/Vendicated/Vencord)

![image](https://user-images.githubusercontent.com/45497981/197824700-5c77bcf3-f8e8-4b5f-95e8-76a17cd40d85.png)

## Usage

### Windows

Download the latest VencordInstaller.exe from [GitHub Releases](https://github.com/Vendicated/VencordInstaller/releases/latest) and run it

### Linux

```sh
sh -c "$(curl -s https://raw.githubusercontent.com/Vendicated/VencordInstaller/main/install.sh)"
```

### Mac

Download the latest Mac build from [GitHub Releases](https://github.com/Vendicated/VencordInstaller/releases/latest), unzip it, and run `VencordInstaller.app` 

If you get a `VencordInstaller can't be opened` message, right-click `VencordInstaller.app` and click open.

This error shows because the app isn't signed, since I'm not paying 100 bucks for an Apple Developer license. 

### Building from source

You need the go compiler and gcc.

On Linux, you also need some dependencies:

```sh
apt install -y pkg-config libsdl2-dev libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libglx-dev libgl1-mesa-dev libxxf86vm-dev
```

Then just run

```sh
go build
```

You might want to pass some flags to this command to get a better build.
See [the GitHub workflow](https://github.com/Vendicated/VencordInstaller/blob/main/.github/workflows/release.yml) for what flags I pass or if you want more precise instructions
