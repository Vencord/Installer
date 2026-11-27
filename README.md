# Vencord Installer

The Vencord Installer allows you to install [Vencord, the cutest Discord Desktop client mod](https://vencord.dev)

<img width="1777" height="1144" alt="Screenshot From 2026-11-27 17-57-05" src="https://github.com/user-attachments/assets/8d439d1f-b6c9-40f0-9008-e0d8fddf8a44" />

## Usage

See https://vencord.dev/download

## Building from source

### Prerequisites 

You need to install the [Go programming language](https://go.dev/doc/install) and GCC, the GNU Compiler Collection (MinGW on Windows)

<details>
<summary>Additionally, if you're using Linux, you have to install some additional dependencies:</summary>

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

Building with `make` is preferred, as it sets some variables and passes other useful build flags. But if you prefer not installing `make`, you can also build without it.

#### Build the GUI

- With make: `make GUI=1`
- Without make: `go build`

#### Build the CLI

- With make: `make`
- Without make: `go build -tags cli`
