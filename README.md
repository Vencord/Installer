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

The Linux GUI uses the x86-64 GLFW and ImGui libraries bundled with cimgui-go.

```sh
apt install -y libgl-dev libx11-dev
dnf install libglvnd-devel libX11-devel
```

</details>

### Building

#### Install dependencies

```sh
go mod tidy
```

#### Build the GUI

On Linux the same binary selects X11 or Wayland at startup.

```sh
make GUI=1
```

#### Build the CLI
```sh
make
```

You might want to pass some flags to this command to get a better build.
See [the GitHub workflow](https://github.com/Vendicated/VencordInstaller/blob/main/.github/workflows/release.yml) for what flags I pass or if you want more precise instructions
