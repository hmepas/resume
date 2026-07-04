#!/bin/sh
set -eu

repo="${RESUME_REPO:-hmepas/resume}"
version="${RESUME_VERSION:-latest}"
install_dir="${INSTALL_DIR:-$HOME/.local/bin}"

os="$(uname -s)"
arch="$(uname -m)"

case "$os" in
  Darwin|Linux) ;;
  *) echo "resume: unsupported OS: $os" >&2; exit 1 ;;
esac

case "$arch" in
  x86_64|amd64) arch="x86_64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "resume: unsupported architecture: $arch" >&2; exit 1 ;;
esac

asset="resume_${os}_${arch}.tar.gz"
base="https://github.com/${repo}/releases"
if [ "$version" = "latest" ]; then
  url="${base}/latest/download/${asset}"
  sums_url="${base}/latest/download/checksums.txt"
else
  url="${base}/download/${version}/${asset}"
  sums_url="${base}/download/${version}/checksums.txt"
fi

tmp="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp"
}
trap cleanup EXIT INT TERM

echo "Downloading ${asset} from ${repo}..."
curl -fsSL "$url" -o "$tmp/$asset"
curl -fsSL "$sums_url" -o "$tmp/checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then
  (cd "$tmp" && grep "  ${asset}\$" checksums.txt | sha256sum -c -)
elif command -v shasum >/dev/null 2>&1; then
  expected="$(grep "  ${asset}\$" "$tmp/checksums.txt" | awk '{print $1}')"
  actual="$(shasum -a 256 "$tmp/$asset" | awk '{print $1}')"
  if [ "$expected" != "$actual" ]; then
    echo "resume: checksum mismatch" >&2
    exit 1
  fi
else
  echo "resume: warning: no sha256 checker found; skipping checksum verification" >&2
fi

tar -xzf "$tmp/$asset" -C "$tmp"
mkdir -p "$install_dir"
install "$tmp/resume" "$install_dir/resume"

echo "Installed resume to $install_dir/resume"
if ! command -v resume >/dev/null 2>&1; then
  echo "Add this to PATH if needed: export PATH=\"$install_dir:\$PATH\""
fi
