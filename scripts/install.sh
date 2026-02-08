#!/usr/bin/env bash
#
# Mailbeads (mb) installation script
# Usage: curl -fsSL https://raw.githubusercontent.com/daviddao/mailbeads/main/scripts/install.sh | bash
#

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()    { echo -e "${BLUE}==>${NC} $1"; }
log_success() { echo -e "${GREEN}==>${NC} $1"; }
log_warning() { echo -e "${YELLOW}==>${NC} $1"; }
log_error()   { echo -e "${RED}Error:${NC} $1" >&2; }

detect_platform() {
    local os arch

    case "$(uname -s)" in
        Darwin) os="darwin" ;;
        Linux)  os="linux" ;;
        *) log_error "Unsupported OS: $(uname -s)"; exit 1 ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64)    arch="amd64" ;;
        aarch64|arm64)   arch="arm64" ;;
        *) log_error "Unsupported architecture: $(uname -m)"; exit 1 ;;
    esac

    echo "${os}_${arch}"
}

check_go() {
    if command -v go &> /dev/null; then
        local go_version
        go_version=$(go version | awk '{print $3}' | sed 's/go//')
        local minor
        minor=$(echo "$go_version" | cut -d. -f2)
        if [ "$minor" -ge 21 ]; then
            return 0
        fi
        log_warning "Go $go_version found but 1.21+ required"
        return 1
    fi
    return 1
}

resign_for_macos() {
    local binary_path=$1
    [[ "$(uname -s)" != "Darwin" ]] && return 0
    command -v codesign &> /dev/null || return 0
    log_info "Re-signing binary for macOS..."
    codesign --remove-signature "$binary_path" 2>/dev/null || true
    codesign --force --sign - "$binary_path" 2>/dev/null || true
}

install_from_release() {
    log_info "Checking for pre-built release..."

    local platform=$1
    local tmp_dir
    tmp_dir=$(mktemp -d)

    local latest_url="https://api.github.com/repos/daviddao/mailbeads/releases/latest"
    local release_json version

    if command -v curl &> /dev/null; then
        release_json=$(curl -fsSL "$latest_url" 2>/dev/null) || return 1
    elif command -v wget &> /dev/null; then
        release_json=$(wget -qO- "$latest_url" 2>/dev/null) || return 1
    else
        return 1
    fi

    version=$(echo "$release_json" | grep '"tag_name"' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')
    [ -z "$version" ] && return 1

    local archive_name="mailbeads_${version#v}_${platform}.tar.gz"
    echo "$release_json" | grep -Fq "\"name\": \"$archive_name\"" || return 1

    local download_url="https://github.com/daviddao/mailbeads/releases/download/${version}/${archive_name}"

    log_info "Downloading $archive_name..."
    cd "$tmp_dir"
    if command -v curl &> /dev/null; then
        curl -fsSL -o "$archive_name" "$download_url" || { cd - > /dev/null; rm -rf "$tmp_dir"; return 1; }
    else
        wget -q -O "$archive_name" "$download_url" || { cd - > /dev/null; rm -rf "$tmp_dir"; return 1; }
    fi

    tar -xzf "$archive_name" || { cd - > /dev/null; rm -rf "$tmp_dir"; return 1; }

    local install_dir
    if [[ -w /usr/local/bin ]]; then
        install_dir="/usr/local/bin"
    else
        install_dir="$HOME/.local/bin"
        mkdir -p "$install_dir"
    fi

    log_info "Installing to $install_dir..."
    if [[ -w "$install_dir" ]]; then
        mv mb "$install_dir/"
    else
        sudo mv mb "$install_dir/"
    fi

    resign_for_macos "$install_dir/mb"
    log_success "mb installed to $install_dir/mb"

    if [[ ":$PATH:" != *":$install_dir:"* ]]; then
        log_warning "$install_dir is not in your PATH"
        echo ""
        echo "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo "  export PATH=\"\$PATH:$install_dir\""
        echo ""
    fi

    cd - > /dev/null
    rm -rf "$tmp_dir"
    return 0
}

install_with_go() {
    log_info "Installing mb using 'go install'..."

    if go install github.com/daviddao/mailbeads/cmd/mb@latest; then
        local gobin
        gobin=$(go env GOBIN 2>/dev/null || true)
        if [ -z "$gobin" ]; then
            gobin="$(go env GOPATH)/bin"
        fi

        resign_for_macos "$gobin/mb"
        log_success "mb installed via go install"

        if [[ ":$PATH:" != *":$gobin:"* ]]; then
            log_warning "$gobin is not in your PATH"
            echo ""
            echo "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
            echo "  export PATH=\"\$PATH:$gobin\""
            echo ""
        fi

        return 0
    fi

    log_error "go install failed"
    return 1
}

build_from_source() {
    log_info "Building mb from source..."

    local tmp_dir
    tmp_dir=$(mktemp -d)
    cd "$tmp_dir"

    git clone --depth 1 https://github.com/daviddao/mailbeads.git || { cd - > /dev/null; rm -rf "$tmp_dir"; return 1; }
    cd mailbeads

    go build -o mb ./cmd/mb || { cd - > /dev/null; rm -rf "$tmp_dir"; return 1; }

    local install_dir
    if [[ -w /usr/local/bin ]]; then
        install_dir="/usr/local/bin"
    else
        install_dir="$HOME/.local/bin"
        mkdir -p "$install_dir"
    fi

    if [[ -w "$install_dir" ]]; then
        mv mb "$install_dir/"
    else
        sudo mv mb "$install_dir/"
    fi

    resign_for_macos "$install_dir/mb"
    log_success "mb installed to $install_dir/mb"

    cd - > /dev/null
    rm -rf "$tmp_dir"
    return 0
}

verify_installation() {
    if command -v mb &> /dev/null; then
        log_success "mb is installed and ready!"
        echo ""
        mb version 2>/dev/null || echo "mb (development build)"
        echo ""
        echo "Get started:"
        echo "  cd your-project"
        echo "  mb init"
        echo "  mb quickstart"
        echo ""
        return 0
    fi

    log_error "mb was installed but is not in PATH"
    return 1
}

main() {
    echo ""
    echo "Mailbeads (mb) Installer"
    echo ""

    log_info "Detecting platform..."
    local platform
    platform=$(detect_platform)
    log_info "Platform: $platform"

    # Try pre-built release first
    if install_from_release "$platform"; then
        verify_installation
        exit 0
    fi

    log_warning "No pre-built release found, trying Go install..."

    # Try go install
    if check_go; then
        if install_with_go; then
            verify_installation
            exit 0
        fi
    fi

    # Build from source
    log_warning "Falling back to building from source..."

    if ! check_go; then
        log_error "Go 1.21+ is required to build from source"
        echo ""
        echo "Install Go from https://go.dev/dl/ then run this script again."
        echo ""
        echo "Or install manually:"
        echo "  go install github.com/daviddao/mailbeads/cmd/mb@latest"
        echo ""
        exit 1
    fi

    if build_from_source; then
        verify_installation
        exit 0
    fi

    log_error "Installation failed"
    echo ""
    echo "Manual installation:"
    echo "  go install github.com/daviddao/mailbeads/cmd/mb@latest"
    echo ""
    exit 1
}

main "$@"
