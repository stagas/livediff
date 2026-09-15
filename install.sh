#!/bin/sh

set -eu

repository="stagas/livediff"
release_url="https://github.com/$repository/releases/latest/download"

case "$(uname -s)" in
  Linux) platform="linux"; executable="livediff" ;;
  Darwin) platform="macos"; executable="livediff" ;;
  MINGW*|MSYS*|CYGWIN*) platform="windows"; executable="livediff.exe" ;;
  *) echo "livediff: unsupported operating system: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) architecture="x64" ;;
  arm64|aarch64) architecture="arm64" ;;
  *) echo "livediff: unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

if [ "$platform" = "windows" ] && [ "$architecture" != "x64" ]; then
  echo "livediff: Windows ARM64 builds are not currently available" >&2
  exit 1
fi

asset="livediff-$platform-$architecture"
if [ "$platform" = "windows" ]; then asset="$asset.exe"; fi

if [ -n "${INSTALL_DIR:-}" ]; then
  install_dir="$INSTALL_DIR"
elif [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
  install_dir="/usr/local/bin"
else
  install_dir="${HOME:?}/.local/bin"
fi

mkdir -p "$install_dir"
temporary_file=$(mktemp "${TMPDIR:-/tmp}/livediff.XXXXXX")
trap 'rm -f "$temporary_file"' EXIT HUP INT TERM

echo "Downloading $asset..."
curl -fL "$release_url/$asset" -o "$temporary_file"
chmod +x "$temporary_file"
mv "$temporary_file" "$install_dir/$executable"
trap - EXIT HUP INT TERM

echo "Installed livediff to $install_dir/$executable"
case ":${PATH:-}:" in
  *":$install_dir:"*) ;;
  *) echo "Add $install_dir to your PATH to run livediff from anywhere." ;;
esac
