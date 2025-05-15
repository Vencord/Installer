#!/bin/sh
set -e

if [ "$(id -u)" -eq 0 ]; then
    echo "Run me as normal user, not root!"
    exit 1
fi

outfile=$(mktemp --tmpdir="$HOME")
trap 'rm -f "$outfile"' EXIT

echo "Downloading Installer..."

set -- "XDG_CONFIG_HOME=$XDG_CONFIG_HOME"

curl -sS https://github.com/Vendicated/VencordInstaller/releases/latest/download/VencordInstallerCli-Linux \
  --output "$outfile" \
  --location \
  --fail

chmod +x "$outfile"

if command -v sudo >/dev/null; then
  echo "Running with sudo"
  sudo env "$@" "$outfile"
elif command -v doas >/dev/null; then
  echo "Running with doas"
  doas env "$@" "$outfile"
elif command -v run0 >/dev/null; then
  echo "Running with run0"
  run0 env "$@" "$outfile"
elif command -v pkexec >/dev/null; then
  echo "Running with pkexec"
  pkexec env "$@" "SUDO_USER=$(whoami)" "$outfile"
else
  echo "Neither sudo nor doas were found. Please install either of them to proceed."
fi
