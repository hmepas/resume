#!/bin/sh

# usage example:
# GOCACHE=.gocache GOMODCACHE=.gomodcache VERSION=v0.1.1 scripts/build-release.sh

set -eu

version="${VERSION:-dev}"
dist="${DIST_DIR:-dist}"

rm -rf "$dist"
mkdir -p "$dist"

build_one() {
  asset_os="$1"
  goos="$2"
  arch="$3"
  goarch="$4"
  out_dir="$dist/resume_${asset_os}_${arch}"
  mkdir -p "$out_dir"

  echo "Building ${goos}/${goarch}..."
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build \
    -ldflags "-s -w -X main.version=${version}" \
    -o "$out_dir/resume" ./cmd/resume

  cp README.md LICENSE "$out_dir/"
  tar -C "$out_dir" -czf "$dist/resume_${asset_os}_${arch}.tar.gz" resume README.md LICENSE
}

build_one Darwin darwin arm64 arm64
build_one Darwin darwin x86_64 amd64
build_one Linux linux arm64 arm64
build_one Linux linux x86_64 amd64

(
  cd "$dist"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum resume_*.tar.gz > checksums.txt
  else
    shasum -a 256 resume_*.tar.gz > checksums.txt
  fi
)

echo "Release artifacts written to ${dist}/"
