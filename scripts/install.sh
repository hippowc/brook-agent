#!/usr/bin/env bash
# 一键安装 Brook：优先从 GitHub Release 下载；失败则用 Go 安装（需网络与 Go 工具链）。
#   curl -fsSL https://raw.githubusercontent.com/hippowc/brook/main/scripts/install.sh | bash
# 环境变量：
#   VERSION=v0.1.0          release 标签；不设置则取 GitHub latest
#   INSTALL_DIR=...         安装目录（默认 /usr/local/bin 或可写则用 ~/.local/bin）
#   BROOK_FORCE_SOURCE=1    跳过 Release，直接 go install
set -euo pipefail

REPO="hippowc/brook"
MODULE="github.com/${REPO}"

detect_plat() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$arch" in
    x86_64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) echo "unsupported arch: $(uname -m)" >&2; exit 1 ;;
  esac
  case "$os" in
    darwin|linux) ;;
    *) echo "unsupported OS: $os (need darwin or linux)" >&2; exit 1 ;;
  esac
  echo "${os}_${arch}"
}

pick_dest() {
  if [[ -n "${INSTALL_DIR:-}" ]]; then
    mkdir -p "$INSTALL_DIR"
    echo "$INSTALL_DIR"
    return
  fi
  if [[ -d /usr/local/bin ]] && [[ -w /usr/local/bin ]]; then
    echo "/usr/local/bin"
    return
  fi
  mkdir -p "${HOME}/.local/bin"
  echo "${HOME}/.local/bin"
}

install_release() {
  local tag="$1"
  local plat="$2"
  local dest="$3"
  local tarball="brook_${tag}_${plat}.tar.gz"
  local url="https://github.com/${REPO}/releases/download/${tag}/${tarball}"

  echo "Trying ${url}"
  local tmp
  tmp="$(mktemp)"
  if ! curl -fsSL -o "$tmp" "$url"; then
    rm -f "$tmp"
    return 1
  fi

  local dir
  dir="$(mktemp -d)"
  tar -xzf "$tmp" -C "$dir"
  rm -f "$tmp"

  local inner="${dir}/brook_${tag}_${plat}"
  if [[ ! -d "$inner" ]]; then
    inner="$(find "$dir" -maxdepth 1 -type d ! -path "$dir" | head -1)"
  fi
  install -m 0755 "${inner}/brook" "${dest}/brook"
  install -m 0755 "${inner}/brook-tui" "${dest}/brook-tui"
  rm -rf "$dir"
  echo "Installed brook + brook-tui -> ${dest}"
  return 0
}

install_go() {
  command -v go >/dev/null 2>&1 || {
    echo "Need Go: https://go.dev/dl/  or publish a GitHub Release with binaries." >&2
    exit 1
  }
  local ref="${VERSION:-latest}"
  echo "go install (${ref}) ..."
  go install "${MODULE}/cmd/brook@${ref}"
  go install "${MODULE}/cmd/brook-tui@${ref}"
  echo "Done. Ensure \$(go env GOPATH)/bin is on PATH."
}

main() {
  local dest
  dest="$(pick_dest)"
  export PATH="${dest}:${PATH}"

  if [[ "${BROOK_FORCE_SOURCE:-}" == "1" ]]; then
    install_go
    exit 0
  fi

  local plat
  plat="$(detect_plat)"

  local tag="${VERSION:-}"
  if [[ -z "$tag" ]]; then
    tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)"
  fi

  if [[ -n "$tag" && "$tag" != "null" ]]; then
    if install_release "$tag" "$plat" "$dest"; then
      exit 0
    fi
  fi

  echo "Release download unavailable; using Go toolchain ..."
  install_go
}

main "$@"
