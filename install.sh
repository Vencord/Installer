#!/bin/sh
set -e

outfile=$(mktemp)
# shellcheck disable=SC2064
trap "rm -rf '$outfile'" EXIT

echo "Downloading Installer..."

rootenv=
kind=wayland
if [ -z "$WAYLAND_DISPLAY" ]; then
  echo "X11 detected"
  kind=x11
else
  echo "Wayland detected"
  rootenv="XDG_RUNTIME_DIR=$XDG_RUNTIME_DIR WAYLAND_DISPLAY=$XDG_RUNTIME_DIR/$WAYLAND_DISPLAY"
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
    sudo env $rootenv "$outfile"
  elif command -v doas >/dev/null; then
    echo "Running with doas"
    doas env $rootenv "$outfile"
  else
    echo "Didn't find sudo or doas, falling back to su"
    su -c "SUDO_USER=$(whoami) '$outfile'"
  fi
else
  echo "Running unprivileged"
  "$outfile"
fi

