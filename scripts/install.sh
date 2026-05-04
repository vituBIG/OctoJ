#!/usr/bin/env bash
# OctoJ installer for Linux and macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/OctavoBit/octoj/main/scripts/install.sh | bash

set -euo pipefail

REPO="OctavoBit/octoj"
BINARY="octoj"
INSTALL_DIR="${OCTOJ_INSTALL_DIR:-}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

log_info()    { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; }
log_section() { echo -e "\n${BOLD}${BLUE}==> $*${NC}"; }

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       log_error "Unsupported OS: $(uname -s)"; exit 1 ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) log_error "Unsupported architecture: $(uname -m)"; exit 1 ;;
    esac
}

# Get the latest release version from GitHub
get_latest_version() {
    local api_url="https://api.github.com/repos/${REPO}/releases/latest"
    local version

    if command -v curl >/dev/null 2>&1; then
        version=$(curl -fsSL "$api_url" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        version=$(wget -qO- "$api_url" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    else
        log_error "Neither curl nor wget is available. Please install one and retry."
        exit 1
    fi

    if [ -z "$version" ]; then
        log_error "Could not determine latest version. Check your internet connection."
        exit 1
    fi

    echo "$version"
}

# Download a file
download() {
    local url="$1"
    local dest="$2"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL --progress-bar -o "$dest" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -q --show-progress -O "$dest" "$url"
    else
        log_error "Neither curl nor wget found."
        exit 1
    fi
}

# Determine install directory
choose_install_dir() {
    if [ -n "$INSTALL_DIR" ]; then
        echo "$INSTALL_DIR"
        return
    fi

    # Prefer /usr/local/bin if writable
    if [ -w "/usr/local/bin" ]; then
        echo "/usr/local/bin"
        return
    fi

    # Fall back to ~/.local/bin
    local local_bin="$HOME/.local/bin"
    mkdir -p "$local_bin"
    echo "$local_bin"
}

main() {
    echo ""
    echo -e "${BOLD}  ___  ___ _        _ ${NC}"
    echo -e "${BOLD} / _ \/ __| |_ ___ | |${NC}"
    echo -e "${BOLD}| (_) \__ \  _/ _ \| |${NC}"
    echo -e "${BOLD} \___/|___/\__\___/|_|  by OctavoBit${NC}"
    echo ""
    echo "  Java JDK Version Manager"
    echo ""

    log_section "Detecting platform"
    OS=$(detect_os)
    ARCH=$(detect_arch)
    log_info "OS: $OS, Architecture: $ARCH"

    log_section "Getting latest version"
    VERSION=$(get_latest_version)
    log_info "Latest version: $VERSION"

    ASSET_NAME="${BINARY}-${OS}-${ARCH}"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET_NAME}"
    CHECKSUM_URL="${DOWNLOAD_URL}.sha256"

    log_section "Downloading OctoJ ${VERSION}"
    log_info "URL: $DOWNLOAD_URL"

    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    BINARY_PATH="${TMP_DIR}/${ASSET_NAME}"
    download "$DOWNLOAD_URL" "$BINARY_PATH"

    # Verify checksum if sha256sum/shasum is available
    if command -v sha256sum >/dev/null 2>&1 || command -v shasum >/dev/null 2>&1; then
        log_section "Verifying checksum"
        CHECKSUM_PATH="${TMP_DIR}/checksum.sha256"
        if download "$CHECKSUM_URL" "$CHECKSUM_PATH" 2>/dev/null; then
            EXPECTED=$(awk '{print $1}' "$CHECKSUM_PATH")
            if command -v sha256sum >/dev/null 2>&1; then
                ACTUAL=$(sha256sum "$BINARY_PATH" | awk '{print $1}')
            else
                ACTUAL=$(shasum -a 256 "$BINARY_PATH" | awk '{print $1}')
            fi

            if [ "$EXPECTED" = "$ACTUAL" ]; then
                log_info "Checksum OK"
            else
                log_error "Checksum mismatch!"
                log_error "  Expected: $EXPECTED"
                log_error "  Actual:   $ACTUAL"
                exit 1
            fi
        else
            log_warn "Could not download checksum file, skipping verification"
        fi
    fi

    log_section "Installing OctoJ"
    INSTALL_DIR=$(choose_install_dir)
    INSTALL_PATH="${INSTALL_DIR}/${BINARY}"

    chmod +x "$BINARY_PATH"

    if [ -w "$INSTALL_DIR" ]; then
        mv "$BINARY_PATH" "$INSTALL_PATH"
    else
        log_info "Installing to $INSTALL_PATH (sudo required)"
        sudo mv "$BINARY_PATH" "$INSTALL_PATH"
    fi

    log_info "Installed to: $INSTALL_PATH"

    log_section "Configuring environment"
    "$INSTALL_PATH" init --apply || {
        log_warn "Automatic environment setup failed. Run 'octoj init --apply' manually."
    }

    # Check if install dir is in PATH
    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        echo ""
        log_warn "Add $INSTALL_DIR to your PATH:"
        echo ""
        echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
        echo ""
        echo "Then restart your terminal or run: source ~/.bashrc (or ~/.zshrc)"
    fi

    echo ""
    log_info "${GREEN}OctoJ ${VERSION} installed successfully!${NC}"
    echo ""
    echo "Get started:"
    echo "  octoj search 21          # search for JDK 21"
    echo "  octoj install 21         # install Temurin JDK 21"
    echo "  octoj use temurin@21     # activate JDK 21"
    echo "  octoj doctor             # check installation"
    echo ""
    echo "Documentation: https://github.com/${REPO}#readme"
    echo ""
}

main "$@"
