#!/bin/sh
set -e

outfile=$(mktemp)
# shellcheck disable=SC2064
trap "rm -rf '$outfile'" EXIT

echo "Downloading Installer..."
echo

curl -sS https://github.com/Vendicated/VencordInstaller/releases/latest/download/VencordInstaller.linux \
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

if [ "$opt" = y ] || [ "$opt" = yes ]; then
  if command -v sudo >/dev/null; then
    echo "Running with sudo"
    sudo "$outfile"
  elif command -v doas >/dev/null; then
    echo "Running with doas"
    doas sh -c "SUDO_USER=$(whoami) '$outfile'"
  else
    echo "Didn't find sudo or doas, falling back to su"
    su -c "SUDO_USER=$(whoami) '$outfile'"
  fi
else
  echo "Running unprivileged"
  "$outfile"
fi

