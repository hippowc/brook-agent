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

# API 请求保持较短超时；Release 二进制可能较大，慢网下 10 分钟仍下不完，故下载不设总时长上限（max-time 0），仅限制连接建立时间。
CURL_API=(curl -fsSL --connect-timeout 15 --max-time 60)
CURL_DL=(curl -fsSL --connect-timeout 45 --max-time 0 --retry 5 --retry-delay 8 --retry-connrefused)

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

  echo "Downloading (慢网可能较久，可设 HTTPS_PROXY): ${url}"
  local tmp
  tmp="$(mktemp)"
  if ! "${CURL_DL[@]}" -o "$tmp" "$url"; then
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
  echo "go install (${ref}) — 首次会拉取依赖，可能持续数分钟且无新输出，请稍候 ..."
  go install "${MODULE}/cmd/brook@${ref}"
  go install "${MODULE}/cmd/brook-tui@${ref}"
  echo "Done. Ensure \$(go env GOPATH)/bin is on PATH."
}

main() {
  local dest plat tag api_json
  dest="$(pick_dest)"
  export PATH="${dest}:${PATH}"

  if [[ "${BROOK_FORCE_SOURCE:-}" == "1" ]]; then
    install_go
    exit 0
  fi

  plat="$(detect_plat)"

  tag="${VERSION:-}"
  if [[ -z "$tag" ]]; then
    echo "Fetching latest release tag from GitHub API ..."
    api_json=""
    api_json="$("${CURL_API[@]}" "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null)" || true
    tag="$(printf '%s' "$api_json" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)"
    if [[ -z "$tag" ]]; then
      echo "提示: 无法访问 GitHub API（超时或网络限制）。可设置 VERSION=v0.0.1 重试，或 export HTTPS_PROXY=... 后再执行。" >&2
    fi
  fi

  if [[ -n "$tag" && "$tag" != "null" ]]; then
    if install_release "$tag" "$plat" "$dest"; then
      exit 0
    fi
  fi

  echo "Release 二进制不可用，改用 Go 源码安装（若仍很慢，请检查网络或代理）..."
  install_go
}

main "$@"
