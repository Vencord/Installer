# Vencord Installer

Vencord Installer allows you to install [Vencord, a Discord Desktop client mod](https://github.com/Vendicated/Vencord)

![image](https://user-images.githubusercontent.com/45497981/226734476-5fb42420-844d-4e27-ae06-4799118e086e.png)

## Usage

### Windows

> **Warning**
**Do not** run the installer as Admin

Download [VencordInstaller.exe](https://github.com/Vencord/Installer/releases/latest/download/VencordInstaller.exe) and run it

Alternatively, or if the above doesn't work for you, open Powershell and run
```ps1
iwr "https://raw.githubusercontent.com/Vencord/Installer/main/install.ps1" -UseBasicParsing | iex
```

### Linux

```sh
sh -c "$(curl -sS https://raw.githubusercontent.com/Vendicated/VencordInstaller/main/install.sh)"
```

### macOS

Download the latest [macOS build](https://github.com/Vencord/Installer/releases/latest/download/VencordInstaller.MacOS.zip), unzip it, and run `VencordInstaller.app` 

If you get a `VencordInstaller can't be opened` warning, right-click `VencordInstaller.app` and click open.

This warning shows because the app isn't signed since I'm not willing to pay 100 bucks a year for an Apple Developer license.

___

### Building from source

You need the go compiler

To build the gui (you can build the cli without these), you also need gcc (MinGW on Windows) and the following additional dependencies if on Linux:

```sh
apt install -y pkg-config libsdl2-dev libglx-dev libgl1-mesa-dev

# X11
apt install -y xorg-dev

# Wayland
apt install -y libwayland-dev libxkbcommon-dev wayland-protocols extra-cmake-modules
```

Then just run

```sh
go mod tidy

# Windows / MacOs / Linux X11
go build
# or build the Cli
go build --tags cli

# Linux Wayland
go build --tags wayland
```

You might want to pass some flags to this command to get a better build.
See [the GitHub workflow](https://github.com/Vendicated/VencordInstaller/blob/main/.github/workflows/release.yml) for what flags I pass or if you want more precise instructions
