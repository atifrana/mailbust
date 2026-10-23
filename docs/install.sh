#!/bin/sh
# MailBust installer for macOS and Linux.
#   curl -sSL https://raw.githubusercontent.com/atifrana/mailbust/main/docs/install.sh | sh
#
# Env overrides:
#   MAILBUST_INSTALL_DIR   install location (default: /usr/local/bin, else ~/.local/bin)
#   MAILBUST_VERSION       version tag to install (default: latest)
set -eu

REPO="atifrana/mailbust"
BIN="mailbust"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  aarch64 | arm64) arch="arm64" ;;
  *) echo "mailbust: unsupported architecture: $arch" >&2; exit 1 ;;
esac
case "$os" in
  linux | darwin) ;;
  *) echo "mailbust: unsupported OS: $os (use install.ps1 on Windows)" >&2; exit 1 ;;
esac

version="${MAILBUST_VERSION:-latest}"
if [ "$version" = "latest" ]; then
  base="https://github.com/$REPO/releases/latest/download"
else
  base="https://github.com/$REPO/releases/download/$version"
fi
url="$base/mailbust_${os}_${arch}.tar.gz"

dir="${MAILBUST_INSTALL_DIR:-}"
if [ -z "$dir" ]; then
  if [ -w /usr/local/bin ] 2>/dev/null; then dir="/usr/local/bin"; else dir="$HOME/.local/bin"; fi
fi
mkdir -p "$dir"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $url"
curl -sSLf "$url" -o "$tmp/m.tar.gz"
tar -xzf "$tmp/m.tar.gz" -C "$tmp"
install -m 0755 "$tmp/$BIN" "$dir/$BIN"

echo "Installed $BIN to $dir/$BIN"
case ":$PATH:" in
  *":$dir:"*) ;;
  *) echo "Note: $dir is not on your PATH. Add it, e.g.: export PATH=\"$dir:\$PATH\"" ;;
esac
"$dir/$BIN" version
