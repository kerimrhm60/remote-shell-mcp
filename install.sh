#!/usr/bin/env bash
# install.sh — fetch the latest remote-shell-mcp release for this OS/arch,
# place the two binaries on PATH, and optionally wire them into detected
# MCP clients (Claude Code, Claude Desktop, Codex CLI).
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/jaenster/remote-shell-mcp/main/install.sh | sh
#   curl -fsSL https://.../install.sh | sh -s -- --no-setup       # skip the MCP-client wiring
#   curl -fsSL https://.../install.sh | sh -s -- --version v0.1.0 # pin a release
#   curl -fsSL https://.../install.sh | sh -s -- --dir /usr/local/bin

set -euo pipefail

REPO="jaenster/remote-shell-mcp"
VERSION="latest"
INSTALL_DIR=""
RUN_SETUP=1
SETUP_YES=0
# BASE_URL overrides the GitHub release URL prefix. With it, the asset is
# fetched from "$BASE_URL/$asset" instead of the github.com release page.
# CI uses this to point the installer at locally-built snapshot artifacts;
# end users should never need it.
BASE_URL=""

while [ "${1-}" != "" ]; do
  case "$1" in
    --version)     VERSION="$2"; shift 2 ;;
    --dir)         INSTALL_DIR="$2"; shift 2 ;;
    --base-url)    BASE_URL="$2"; shift 2 ;;
    --no-setup)    RUN_SETUP=0; shift ;;
    --yes|-y)      SETUP_YES=1; shift ;;
    --help|-h)
      sed -n '2,16p' "$0"
      exit 0
      ;;
    *) echo "unknown flag: $1" >&2; exit 2 ;;
  esac
done

# --- detect OS + arch ------------------------------------------------------
uname_s=$(uname -s)
uname_m=$(uname -m)
case "$uname_s" in
  Darwin) os="darwin" ;;
  Linux)  os="linux" ;;
  *)      echo "unsupported OS: $uname_s" >&2; exit 1 ;;
esac
case "$uname_m" in
  arm64|aarch64) arch="arm64" ;;
  x86_64|amd64)  arch="amd64" ;;
  *) echo "unsupported architecture: $uname_m" >&2; exit 1 ;;
esac

# --- pick install dir ------------------------------------------------------
if [ -z "$INSTALL_DIR" ]; then
  if [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="$HOME/.local/bin"
  fi
fi
mkdir -p "$INSTALL_DIR"
echo "Install dir: $INSTALL_DIR"

# --- resolve version -------------------------------------------------------
# Only resolve `latest` against GitHub when BASE_URL isn't overridden. With a
# local base URL we don't have a "latest" concept; the caller passes --version.
if [ -z "$BASE_URL" ] && [ "$VERSION" = "latest" ]; then
  echo "Resolving latest release..."
  redirect=$(curl -fsSL -o /dev/null -w "%{url_effective}" "https://github.com/$REPO/releases/latest")
  VERSION="${redirect##*/}"
  if [ -z "$VERSION" ] || [ "$VERSION" = "latest" ]; then
    echo "could not resolve latest release tag" >&2
    exit 1
  fi
fi
echo "Version: $VERSION"

# --- download + extract ----------------------------------------------------
asset="remote-shell-mcp_${VERSION#v}_${os}_${arch}.tar.gz"
if [ -n "$BASE_URL" ]; then
  prefix="${BASE_URL%/}"
else
  prefix="https://github.com/$REPO/releases/download/$VERSION"
fi
url="$prefix/$asset"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $url"
if ! curl -fsSL -o "$tmp/pkg.tar.gz" "$url"; then
  echo "download failed: $url" >&2
  echo "(verify the release exists at https://github.com/$REPO/releases)" >&2
  exit 1
fi

# Optional checksum verification.
if curl -fsSL -o "$tmp/checksums.txt" "$prefix/checksums.txt" 2>/dev/null; then
  expected=$(grep -F "  $asset" "$tmp/checksums.txt" | awk '{print $1}' || true)
  if [ -n "$expected" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      got=$(sha256sum "$tmp/pkg.tar.gz" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
      got=$(shasum -a 256 "$tmp/pkg.tar.gz" | awk '{print $1}')
    fi
    if [ -n "${got:-}" ] && [ "$got" != "$expected" ]; then
      echo "checksum mismatch! got=$got expected=$expected" >&2
      exit 1
    fi
    echo "Checksum OK"
  fi
fi

tar -xzf "$tmp/pkg.tar.gz" -C "$tmp"

for b in remote-shell-mcpd remote-shell-mcp; do
  if [ ! -f "$tmp/$b" ]; then
    echo "archive missing $b" >&2
    exit 1
  fi
  install -m 0755 "$tmp/$b" "$INSTALL_DIR/$b"
  echo "Installed $INSTALL_DIR/$b"
done

# --- PATH hint -------------------------------------------------------------
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo
    echo "NOTE: $INSTALL_DIR is not on your PATH. Add it to your shell rc:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    echo
    ;;
esac

# --- register with MCP clients ---------------------------------------------
if [ "$RUN_SETUP" -eq 1 ]; then
  echo
  echo "Registering with detected MCP clients..."
  args=""
  if [ "$SETUP_YES" -eq 1 ]; then
    args="--yes"
  fi
  "$INSTALL_DIR/remote-shell-mcp" setup $args || true
fi

echo
echo "Done. Try:  $INSTALL_DIR/remote-shell-mcp version"
