#!/bin/bash
# Installation script for actionsum

set -e

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
REPO="hugo/actionsum"
BINARY_NAME="actionsum"

# Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

if [ "$OS" != "linux" ]; then
    echo "Error: This script only supports Linux"
    exit 1
fi

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Error: Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

echo "Installing actionsum for $OS/$ARCH..."

# Get latest release version
LATEST_VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo "Error: Could not determine latest version"
    exit 1
fi

echo "Latest version: $LATEST_VERSION"

# Download URL
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/${BINARY_NAME}_${LATEST_VERSION#v}_${OS}_${ARCH}.tar.gz"

echo "Downloading from: $DOWNLOAD_URL"

# Download and extract
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

if ! curl -sL "$DOWNLOAD_URL" -o "${BINARY_NAME}.tar.gz"; then
    echo "Error: Failed to download release"
    rm -rf "$TEMP_DIR"
    exit 1
fi

tar -xzf "${BINARY_NAME}.tar.gz"

# Install
echo "Installing to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
    mv "$BINARY_NAME" "$INSTALL_DIR/"
else
    sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
fi

# Cleanup
cd -
rm -rf "$TEMP_DIR"

echo ""
echo "✓ actionsum installed successfully!"
echo ""
echo "Run 'actionsum help' to get started"
