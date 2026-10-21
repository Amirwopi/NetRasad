#!/usr/bin/env bash
# Linux counterpart of build.ps1: builds NetRasad with the version injected.
#
#   ./scripts/build.sh v1.2.3             build/bin/NetRasad
#   ./scripts/build.sh v1.2.3 --package   also build/dist/*.tar.gz, *.deb and *.rpm
#
# Requires the Wails CLI plus the GTK 3 and WebKitGTK 4.1 development files.
# Packaging also requires nfpm:
#   go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
set -euo pipefail

version="${1:-}"
if [[ -z "$version" ]]; then
	echo "Usage: $0 <version> [--package]    e.g. $0 v1.2.3" >&2
	exit 1
fi

cd "$(dirname "$0")/.."

echo "Building NetRasad $version ..."
wails build -clean -platform linux/amd64 -tags webkit2_41 -ldflags "-X main.Version=$version"

if [[ "${2:-}" != "--package" ]]; then
	exit 0
fi

dist=build/dist
rm -rf "$dist"
mkdir -p "$dist"

name="NetRasad-$version-linux-amd64"
staging="$(mktemp -d)"
trap 'rm -rf "$staging"' EXIT
mkdir "$staging/$name"
cp build/bin/NetRasad build/linux/NetRasad.desktop "$staging/$name/"
cp build/appicon.png "$staging/$name/netrasad.png"
tar -C "$staging" -czf "$dist/$name.tar.gz" "$name"

export NETRASAD_VERSION="$version"
nfpm package --config build/linux/nfpm.yaml --packager deb --target "$dist/"
nfpm package --config build/linux/nfpm.yaml --packager rpm --target "$dist/"

echo "Packages written to $dist:"
ls -l "$dist"
