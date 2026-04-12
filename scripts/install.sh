#!/usr/bin/env bash
# 从 GitHub Release 下载预编译二进制并安装到本机 PATH（仅支持 Release 包，不使用 go install）。
#   curl -fsSL https://raw.githubusercontent.com/hippowc/brook/main/scripts/install.sh | bash
# 环境变量：
#   VERSION=v0.1.0          release 标签；不设置则请求 GitHub API 取 latest
#   INSTALL_DIR=...         安装目录（默认 /usr/local/bin 或可写则用 ~/.local/bin）
#   BROOK_BINARIES=...      逗号分隔，要安装的组件名；默认 brook,brook-tui
#                           可选：brook | brook-tui | brook-gateway（可任意组合）
set -euo pipefail

REPO="hippowc/brook"

# API：短超时、静默 JSON。下载：进度条到 stderr、长连接超时、可重试。
CURL_API=(curl -fsSL --connect-timeout 15 --max-time 60)
CURL_DL=(curl -fL --progress-bar --connect-timeout 45 --max-time 0 --retry 5 --retry-delay 8 --retry-connrefused)

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

# 安装单个 Release 包：${bin}_${tag}_${plat}.tar.gz，内含目录 ${bin}_${tag}_${plat}/${bin}
install_one_release() {
  local bin="$1"
  local tag="$2"
  local plat="$3"
  local dest="$4"
  local tarball="${bin}_${tag}_${plat}.tar.gz"
  local url="https://github.com/${REPO}/releases/download/${tag}/${tarball}"

  echo >&2 "==> ${bin} (${tarball})"
  local tmp
  tmp="$(mktemp)"
  if ! "${CURL_DL[@]}" -o "$tmp" "$url"; then
    rm -f "$tmp"
    echo "error: download failed: ${url}" >&2
    return 1
  fi

  local dir
  dir="$(mktemp -d)"
  tar -xzf "$tmp" -C "$dir"
  rm -f "$tmp"

  local inner="${dir}/${bin}_${tag}_${plat}"
  if [[ ! -d "$inner" ]]; then
    inner="$(find "$dir" -maxdepth 1 -type d ! -path "$dir" | head -1)"
  fi
  if [[ ! -f "${inner}/${bin}" ]]; then
    echo "install_one_release: ${inner}/${bin} not found" >&2
    rm -rf "$dir"
    return 1
  fi
  install -m 0755 "${inner}/${bin}" "${dest}/${bin}"
  rm -rf "$dir"
  return 0
}

install_release() {
  local tag="$1"
  local plat="$2"
  local dest="$3"
  local csv="$4"

  IFS=',' read -ra bins <<< "$csv"
  local b
  for b in "${bins[@]}"; do
    b="$(echo "$b" | xargs)"
    if [[ -z "$b" ]]; then
      continue
    fi
    if ! install_one_release "$b" "$tag" "$plat" "$dest"; then
      return 1
    fi
  done
  echo "Installed -> ${dest} : ${csv//,/ }"
  return 0
}

main() {
  local dest plat tag api_json
  dest="$(pick_dest)"
  export PATH="${dest}:${PATH}"

  local csv="${BROOK_BINARIES:-brook,brook-tui}"
  csv="$(echo "$csv" | tr -d '[:space:]')"

  plat="$(detect_plat)"

  tag="${VERSION:-}"
  if [[ -z "$tag" ]]; then
    echo "Fetching latest release tag from GitHub API ..."
    api_json=""
    api_json="$("${CURL_API[@]}" "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null)" || true
    tag="$(printf '%s' "$api_json" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)"
  fi

  if [[ -z "$tag" || "$tag" == "null" ]]; then
    echo "error: 无法确定 Release 版本。请设置 VERSION=v0.x.x 后重试，或检查网络/GitHub API（可 export HTTPS_PROXY=...）。" >&2
    exit 1
  fi

  if ! install_release "$tag" "$plat" "$dest" "$csv"; then
    echo "error: Release 二进制安装失败。请确认 https://github.com/${REPO}/releases/tag/${tag} 已上传对应 ${plat} 的 tar.gz（见 README 发布说明）。" >&2
    exit 1
  fi
}

main "$@"
