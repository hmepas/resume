#!/bin/sh

set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: scripts/bump-homebrew-formula.sh v0.1.2" >&2
  exit 2
fi

version="$1"
case "$version" in
  v*) formula_version="${version#v}" ;;
  *) formula_version="$version"; version="v$version" ;;
esac

formula="packaging/homebrew/resume.rb"
checksums="dist/checksums.txt"

VERSION="$version" scripts/build-release.sh

checksum_for() {
  asset="$1"
  awk -v asset="$asset" '$2 == asset { print $1 }' "$checksums"
}

darwin_arm64="$(checksum_for resume_Darwin_arm64.tar.gz)"
darwin_x86_64="$(checksum_for resume_Darwin_x86_64.tar.gz)"
linux_arm64="$(checksum_for resume_Linux_arm64.tar.gz)"
linux_x86_64="$(checksum_for resume_Linux_x86_64.tar.gz)"

if [ -z "$darwin_arm64" ] || [ -z "$darwin_x86_64" ] || [ -z "$linux_arm64" ] || [ -z "$linux_x86_64" ]; then
  echo "missing one or more checksums in $checksums" >&2
  exit 1
fi

tmp="$(mktemp)"
awk \
  -v version="$formula_version" \
  -v darwin_arm64="$darwin_arm64" \
  -v darwin_x86_64="$darwin_x86_64" \
  -v linux_arm64="$linux_arm64" \
  -v linux_x86_64="$linux_x86_64" '
    /version "/ {
      sub(/version "[^"]+"/, "version \"" version "\"")
    }
    /resume_Darwin_arm64.tar.gz/ {
      print
      getline
      sub(/sha256 "[^"]+"/, "sha256 \"" darwin_arm64 "\"")
    }
    /resume_Darwin_x86_64.tar.gz/ {
      print
      getline
      sub(/sha256 "[^"]+"/, "sha256 \"" darwin_x86_64 "\"")
    }
    /resume_Linux_arm64.tar.gz/ {
      print
      getline
      sub(/sha256 "[^"]+"/, "sha256 \"" linux_arm64 "\"")
    }
    /resume_Linux_x86_64.tar.gz/ {
      print
      getline
      sub(/sha256 "[^"]+"/, "sha256 \"" linux_x86_64 "\"")
    }
    { print }
  ' "$formula" > "$tmp"
mv "$tmp" "$formula"

echo "Updated $formula to $formula_version"
echo
echo "Next:"
echo "  git diff -- packaging/homebrew/resume.rb"
echo "  git status --short"
echo "  git add . && git commit -m 'Release $version' && git push"
