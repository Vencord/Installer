# Vencord Installer

The Vencord Installer allows you to install [Vencord, the cutest Discord Desktop client mod](https://github.com/Vendicated/Vencord)

![image](https://user-images.githubusercontent.com/45497981/226734476-5fb42420-844d-4e27-ae06-4799118e086e.png)

## Usage

See https://vencord.dev/download

## Building from source

### Prerequisites 

You need to install the [Go programming language](https://go.dev/doc/install) and GCC, the GNU Compiler Collection (MinGW on Windows)

<details>
<summary>Additionally, if you're using Linux, you have to install some additional dependencies:</summary>

#### Base dependencies
```sh
apt install -y pkg-config libsdl2-dev libglx-dev libgl1-mesa-dev
dnf install pkg-config libGL-devel libXxf86vm-devel
```

#### X11 dependencies
```sh
apt install -y xorg-dev
dnf install libXcursor-devel libXi-devel libXinerama-devel libXrandr-devel
```

#### Wayland dependencies
```sh
apt install -y libwayland-dev libxkbcommon-dev wayland-protocols extra-cmake-modules
dnf install wayland-devel libxkbcommon-devel wayland-protocols-devel extra-cmake-modules
```

</details>

### Building

#### Install dependencies

```sh
go mod tidy
```

#### Build the GUI

##### Windows / Mac / Linux X11
```sh
go build
```

##### Linux Wayland
```sh
go build --tags wayland
```

#### Build the CLI
```
go build --tags cli
```

You might want to pass some flags to this command to get a better build.
See [the GitHub workflow](https://github.com/Vendicated/VencordInstaller/blob/main/.github/workflows/release.yml) for what flags I pass or if you want more precise instructions

### Notes

#### MacOS
In order to use cli in mac os you need to grant Terminal Full Disk Accesss in *System Settings > Privacy & Security > Full Disk Access*