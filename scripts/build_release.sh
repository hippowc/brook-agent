#!/usr/bin/env bash
# 交叉编译 macOS / Linux 的 release 二进制（brook + brook-tui），输出 dist/ 下的 tar.gz 与 checksums.txt
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

VERSION="${VERSION:-}"
if [[ -z "$VERSION" ]]; then
  VERSION="$(git -C "$ROOT" describe --tags --always 2>/dev/null || true)"
fi
if [[ -z "$VERSION" ]]; then
  VERSION="0.0.0-dev"
fi

export CGO_ENABLED=0
LDFLAGS="-s -w"

DIST="$ROOT/dist"
rm -rf "$DIST"
mkdir -p "$DIST"

platforms=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
)

checksums="$DIST/checksums.txt"
: >"$checksums"

for row in "${platforms[@]}"; do
  # shellcheck disable=SC2086
  set -- $row
  goos="$1"
  goarch="$2"
  name="brook_${VERSION}_${goos}_${goarch}"
  stage="$DIST/${name}"
  mkdir -p "$stage"

  echo "==> GOOS=$goos GOARCH=$goarch -> $name.tar.gz"
  GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags="$LDFLAGS" -o "$stage/brook" ./cmd/brook
  GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags="$LDFLAGS" -o "$stage/brook-tui" ./cmd/brook-tui

  (cd "$DIST" && tar -czf "${name}.tar.gz" "$name")
  rm -rf "$stage"

  (cd "$DIST" && {
    if command -v sha256sum >/dev/null 2>&1; then
      sha256sum "${name}.tar.gz"
    else
      shasum -a 256 "${name}.tar.gz"
    fi
  } >>"$checksums")
done

echo "Done. Artifacts in $DIST"
cat "$checksums"
