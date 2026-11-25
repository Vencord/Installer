#!/usr/bin/env bash
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y --no-install-recommends \
    build-essential ca-certificates cmake git pkg-config libgl1-mesa-dev \
    libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxkbcommon-dev libwayland-dev

export PATH="/usr/local/go/bin:$PATH"
git config --global --add safe.directory "$PWD"
go mod download
cimgui=$(go list -m -f '{{.Dir}}' github.com/AllenDang/cimgui-go)

cmake -S "$cimgui/thirdparty/glfw" -B /tmp/glfw \
    -DGLFW_BUILD_EXAMPLES=OFF -DGLFW_BUILD_TESTS=OFF -DGLFW_BUILD_DOCS=OFF
cmake --build /tmp/glfw --parallel 4

cmake -S "$cimgui/lib" -B /tmp/cimgui
cmake --build /tmp/cimgui --parallel 4

# Replace the upstream archives only in this container's private Go module cache.
cp -f /tmp/glfw/src/libglfw3.a /tmp/cimgui/cimgui.a "$cimgui/lib/linux/x64/"
make GUI=1

required_glibc=$(readelf --version-info build/VencordInstaller-linux | grep -oE 'GLIBC_[0-9.]+' | sort -Vu | tail -1)
echo "Highest required glibc symbol version: $required_glibc"
printf '%s\n' "$required_glibc" GLIBC_2.31 | sort -VC
