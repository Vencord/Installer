#!/bin/sh
set -e

if [ "$(id -u)" -eq 0 ]; then
    echo "Run me as normal user, not root!"
    exit 1
fi

outfile=$(mktemp)
trap 'rm -rf "$outfile"' EXIT

echo "Downloading Installer..."

set -- "XDG_CONFIG_HOME=$XDG_CONFIG_HOME"
kind=wayland
if [ -z "$WAYLAND_DISPLAY" ]; then
  echo "X11 detected"
  kind=x11
else
  echo "Wayland detected"
  set -- "$@" "XDG_RUNTIME_DIR=$XDG_RUNTIME_DIR" "WAYLAND_DISPLAY=$XDG_RUNTIME_DIR/$WAYLAND_DISPLAY"
fi

curl -sS https://github.com/Vendicated/VencordInstaller/releases/latest/download/VencordInstaller-$kind \
  --output "$outfile" \
  --location

chmod +x "$outfile"

echo
echo "Now running VencordInstaller"
echo "Do you want to run as root? [Y|n]"
echo "This is necessary if Discord is in a root owned location like /usr/share or /opt"
printf "> "
read -r runAsRoot

opt="$(echo "$runAsRoot" | tr "[:upper:]" "[:lower:]")"

if [ -z "$opt" ] || [ "$opt" = y ] || [ "$opt" = yes ]; then
  if command -v sudo >/dev/null; then
    echo "Running with sudo"
    sudo env "$@" "$outfile"
  elif command -v doas >/dev/null; then
    echo "Running with doas"
    doas env "$@" "$outfile"
  else
    echo "Didn't find sudo or doas, falling back to su"
    su -c "SUDO_USER=$(whoami) 'env $outfile'"
  fi
else
  echo "Running unprivileged"
  "$outfile"
fi
