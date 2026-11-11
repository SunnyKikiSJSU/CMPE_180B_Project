#!/usr/bin/env bash

# Installs a current Go SDK into $HOME/go-sdk for the Book Metaverse backend.
# Run this inside WSL (Ubuntu), not from PowerShell or cmd.exe.
#
#   bash scripts/install-go-wsl.sh
#   source ~/.bashrc
#   go version

set -Eeuo pipefail

MIN_MINOR=25

if command -v go >/dev/null 2>&1; then
  installed="$(go version | awk '{print $3}')" # e.g. go1.25.4
  minor="${installed#go1.}"
  minor="${minor%%.*}"
  if [[ "$minor" =~ ^[0-9]+$ ]] && ((minor >= MIN_MINOR)); then
    printf 'Go %s is already installed.\n' "$installed"
    exit 0
  fi
  printf 'Found Go %s; installing a current SDK instead.\n' "$installed"
fi

case "$(uname -m)" in
  x86_64)
    goarch="amd64"
    ;;
  aarch64 | arm64)
    goarch="arm64"
    ;;
  *)
    printf 'Unsupported architecture: %s\n' "$(uname -m)" >&2
    exit 1
    ;;
esac

version="$(curl -fsSL 'https://go.dev/VERSION?m=text' | head -n1)"
if [[ -z "$version" ]]; then
  printf 'Could not determine the latest Go version from go.dev.\n' >&2
  exit 1
fi

tarball="${version}.linux-${goarch}.tar.gz"
temp_dir="$(mktemp -d)"
trap 'rm -rf "$temp_dir"' EXIT

printf 'Downloading %s\n' "$tarball"
curl -fsSL "https://go.dev/dl/${tarball}" -o "${temp_dir}/${tarball}"

rm -rf "$HOME/go-sdk"
mkdir -p "$HOME/go-sdk"
tar -C "$HOME/go-sdk" --strip-components=1 -xzf "${temp_dir}/${tarball}"

if ! grep -q 'GOROOT="$HOME/go-sdk"' "$HOME/.bashrc" 2>/dev/null; then
  cat >>"$HOME/.bashrc" <<'EOF'

# Go SDK for the Book Metaverse backend
export GOROOT="$HOME/go-sdk"
export GOPATH="$HOME/go"
export PATH="$GOROOT/bin:$GOPATH/bin:$PATH"
EOF
fi

export GOROOT="$HOME/go-sdk"
export GOPATH="$HOME/go"
export PATH="$GOROOT/bin:$GOPATH/bin:$PATH"

printf 'Installed %s\n' "$(go version)"
printf 'Run: source ~/.bashrc\n'
