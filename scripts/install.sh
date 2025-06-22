#!/usr/bin/env bash

# -------- Logging --------
RED="\e[31m"
YELLOW="\e[33m"
GREEN="\e[32m"
RESET="\e[0m"

log_err()   { echo -e "${RED}[ERROR]${RESET} $*" >&2; }
log_warn()  { echo -e "${YELLOW}[WARN]${RESET} $*" >&2; }
log_info()  { echo -e "${GREEN}[INFO]${RESET} $*"; }

# -------- Spinner --------
spinner() {
  local pid=$1 delay=0.1 spinstr='|/-\'
  while ps -p $pid > /dev/null 2>&1; do
    local temp=${spinstr#?}
    printf " [%c]  " "$spinstr"
    spinstr=$temp${spinstr%"$temp"}
    sleep $delay
    printf "\b\b\b\b\b\b"
  done
  printf "    \b\b\b\b"
}

# -------- Root check --------
is_root() {
  [ "$(id -u)" -eq 0 ] && echo "true" || echo "false"
}

# -------- Dependency check --------
check_dependencies() {
  missing=""
  for cmd in "$@"; do
    command -v "$cmd" >/dev/null 2>&1 || missing="$missing $cmd"
  done
  [ -n "$missing" ] && log_err "Missing dependencies:$missing" && exit 1
}

# -------- Argument parsing --------
BINARIES=("tunman" "tunmand")
parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -o|--output|--install-dir)
        CUSTOM_OUTPUT_DIR="$2"; shift 2 ;;
      -a|--arch|--architecture)
        CUSTOM_ARCH="$2"; shift 2 ;;
      --os|--operating-system)
        CUSTOM_OS="$2"; shift 2 ;;
      -*)
        log_err "Unknown option: $1"; exit 1 ;;
      *)
        log_err "Unknown argument: $1"; exit 1 ;;
    esac
  done
}

# -------- Binary installer --------
install_binary() {
  local BINARY_NAME="$1"
  local INSTALL_DIR="${CUSTOM_OUTPUT_DIR:-/usr/local/bin}"
  local OS="${CUSTOM_OS:-$(uname -s)}"
  local ARCH="${CUSTOM_ARCH:-$(uname -m)}"
  local GITHUB_REPO="Phillezi/tunman2"

  # Normalize OS
  case "$OS" in
    Linux|linux) OS="linux" ;;
    Darwin|macOS|MacOS|Mac|mac) OS="darwin" ;;
    *) log_err "Unsupported OS: $OS"; exit 1 ;;
  esac

  # Normalize ARCH
  case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) log_err "Unsupported architecture: $ARCH"; exit 1 ;;
  esac

  # Skip install if already present (only if no overrides)
  if [ -z "$CUSTOM_OS" ] && [ -z "$CUSTOM_ARCH" ] && [ -z "$CUSTOM_OUTPUT_DIR" ]; then
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
      log_info "$BINARY_NAME is already installed"
      return
    fi
  fi

  # Get release version
  local VERSION=$(curl -s "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | jq -r .tag_name)
  [ "$VERSION" == "null" ] && log_err "Failed to fetch latest version from: ${GITHUB_REPO}" && exit 1

  local URL="https://github.com/${GITHUB_REPO}/releases/download/$VERSION/${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
  local TMP_TAR="/tmp/${BINARY_NAME}.tar.gz"

  log_info "Downloading $BINARY_NAME $VERSION for $OS $ARCH..."
  curl -fSslL -o "$TMP_TAR" "$URL" & CURL_PID=$!
  spinner $CURL_PID
  wait $CURL_PID || { log_err "Failed to download $BINARY_NAME"; exit 1; }

  log_info "Extracting..."
  tar -xf "$TMP_TAR" -C /tmp

  local SUDO=""
  [ "$(is_root)" = "false" ] && SUDO="sudo"
  $SUDO mkdir -p "$INSTALL_DIR"
  $SUDO mv "/tmp/${BINARY_NAME}_${OS}_${ARCH}" "$INSTALL_DIR/$BINARY_NAME"
  $SUDO chmod +x "$INSTALL_DIR/$BINARY_NAME"

  log_info "$BINARY_NAME installed successfully to $INSTALL_DIR"
}

# -------- Main logic --------
main() {
  parse_args "$@"
  check_dependencies curl jq tar

  if [ ${#BINARIES[@]} -eq 0 ]; then
    log_err "No binary specified. Use --bin tunman or --all"
    exit 1
  fi

  for bin in "${BINARIES[@]}"; do
    if [[ ! -z "$bin" ]]; then
      install_binary "$bin" || true
    fi
  done
}

main "$@"
