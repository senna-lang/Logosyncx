#!/usr/bin/env bash
# install.sh — logos installer
#
# Usage:
#   curl -sSfL https://raw.githubusercontent.com/senna-lang/Logosyncx/main/scripts/install.sh | bash
#
# Environment variables:
#   LOGOS_VERSION     Pin a specific release tag (e.g. v0.2.0). Defaults to latest.
#   LOGOS_INSTALL_DIR Override the install directory. Defaults to ~/.local/bin.

set -euo pipefail

REPO="senna-lang/Logosyncx"
BINARY_NAME="logos"
INSTALL_DIR="${LOGOS_INSTALL_DIR:-$HOME/.local/bin}"

# ── helpers ──────────────────────────────────────────────────────────────────

info()  { printf '\033[1;34m[logos]\033[0m %s\n' "$*"; }
ok()    { printf '\033[1;32m[logos]\033[0m %s\n' "$*"; }
warn()  { printf '\033[1;33m[logos]\033[0m %s\n' "$*" >&2; }
fatal() { printf '\033[1;31m[logos] error:\033[0m %s\n' "$*" >&2; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fatal "Required command not found: $1"
}

# ── detect platform ───────────────────────────────────────────────────────────

detect_os() {
  local raw
  raw=$(uname -s)
  case "$raw" in
    Darwin) echo "darwin" ;;
    Linux)  echo "linux"  ;;
    *)      fatal "Unsupported operating system: $raw" ;;
  esac
}

detect_arch() {
  local raw
  raw=$(uname -m)
  case "$raw" in
    x86_64)          echo "amd64" ;;
    aarch64 | arm64) echo "arm64" ;;
    *)               fatal "Unsupported architecture: $raw" ;;
  esac
}

# ── resolve version ───────────────────────────────────────────────────────────

resolve_version() {
  if [ -n "${LOGOS_VERSION:-}" ]; then
    echo "$LOGOS_VERSION"
    return
  fi
  require_cmd curl
  local version
  version=$(
    curl -sSf \
      -H "Accept: application/vnd.github+json" \
      "https://api.github.com/repos/$REPO/releases/latest" \
    | grep '"tag_name"' \
    | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/'
  )
  [ -n "$version" ] || fatal "Could not determine the latest release version. Set LOGOS_VERSION to pin one."
  echo "$version"
}

# ── download helpers ──────────────────────────────────────────────────────────

download() {
  local url="$1" dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -sSfL "$url" -o "$dest"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O "$dest"
  else
    fatal "Neither curl nor wget is available. Please install one and try again."
  fi
}

# ── checksum verification ─────────────────────────────────────────────────────

verify_checksum() {
  local archive="$1" checksum_file="$2" archive_name="$3"

  # Extract expected hash for this archive (supports both "hash  name" and "hash *name")
  local expected
  expected=$(grep -E "(^| )${archive_name}$" "$checksum_file" | awk '{print $1}')
  [ -n "$expected" ] || fatal "Checksum entry for '$archive_name' not found in checksums.txt"

  local actual
  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$archive" | awk '{print $1}')
  elif command -v shasum >/dev/null 2>&1; then
    actual=$(shasum -a 256 "$archive" | awk '{print $1}')
  else
    warn "Cannot verify checksum: neither sha256sum nor shasum is available."
    return
  fi

  if [ "$actual" != "$expected" ]; then
    fatal "Checksum mismatch for $archive_name.\n  expected: $expected\n  actual:   $actual"
  fi
}

# ── extraction ────────────────────────────────────────────────────────────────

extract_binary() {
  local archive="$1" dest_dir="$2"
  case "$archive" in
    *.tar.gz | *.tgz) tar -xzf "$archive" -C "$dest_dir" ;;
    *.zip)            unzip -q "$archive" -d "$dest_dir" ;;
    *)                fatal "Unknown archive format: $archive" ;;
  esac
}

# ── PATH hint ─────────────────────────────────────────────────────────────────

path_hint() {
  local dir="$1"
  # Check each component of PATH
  local found=false
  local IFS=":"
  for p in $PATH; do
    [ "$p" = "$dir" ] && found=true && break
  done
  if [ "$found" = false ]; then
    warn "$dir is not in your \$PATH."
    echo ""
    echo "  Add the following line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo ""
    echo "    export PATH=\"\$PATH:$dir\""
    echo ""
    echo "  Then reload your shell or run:"
    echo ""
    echo "    source ~/.zshrc   # or ~/.bashrc, etc."
    echo ""
  fi
}

# ── main ──────────────────────────────────────────────────────────────────────

main() {
  require_cmd uname

  local os arch version archive archive_name
  os=$(detect_os)
  arch=$(detect_arch)
  version=$(resolve_version)

  archive_name="${BINARY_NAME}_${os}_${arch}.tar.gz"
  local base_url="https://github.com/$REPO/releases/download/$version"
  local archive_url="$base_url/$archive_name"
  local checksum_url="$base_url/checksums.txt"

  info "Installing logos $version ($os/$arch) → $INSTALL_DIR/$BINARY_NAME"

  # Temp workspace
  local tmp_dir
  tmp_dir=$(mktemp -d)
  trap 'rm -rf "$tmp_dir"' EXIT

  local archive_path="$tmp_dir/$archive_name"
  local checksum_path="$tmp_dir/checksums.txt"

  # Download
  info "Downloading $archive_name ..."
  download "$archive_url" "$archive_path"

  info "Downloading checksums.txt ..."
  download "$checksum_url" "$checksum_path"

  # Verify
  info "Verifying checksum ..."
  verify_checksum "$archive_path" "$checksum_path" "$archive_name"

  # Extract
  extract_binary "$archive_path" "$tmp_dir"

  local binary_src="$tmp_dir/$BINARY_NAME"
  [ -f "$binary_src" ] || fatal "Binary '$BINARY_NAME' not found in archive."

  # Install
  mkdir -p "$INSTALL_DIR"
  local binary_dest="$INSTALL_DIR/$BINARY_NAME"

  # If the target already exists and is not writable, try with sudo
  if [ -e "$binary_dest" ] && [ ! -w "$binary_dest" ]; then
    warn "Cannot write to $binary_dest — trying with sudo ..."
    sudo install -m 0755 "$binary_src" "$binary_dest"
  else
    install -m 0755 "$binary_src" "$binary_dest"
  fi

  ok "Installed $binary_dest"
  "$binary_dest" version

  echo ""
  path_hint "$INSTALL_DIR"
}

main "$@"
