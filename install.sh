#!/bin/sh
set -e

# Determine OS and Architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
fi

# Set Version (Match your GitHub Release Tag)
VERSION="v1.1.2"

# Formulate the Binary Name and GitHub Release URL
BINARY_NAME="blsqui-cli-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/blsqui/blsqui-cli/releases/download/${VERSION}/${BINARY_NAME}"

echo "📥 Downloading Blsqui CLI ${VERSION} for ${OS}/${ARCH}..."

# Download Binary to local binary directory
TARGET_DIR="/usr/local/bin"
if [ ! -w "$TARGET_DIR" ]; then
    echo "🔑 Root permissions required to install to $TARGET_DIR. Using sudo..."
    sudo curl -fsSL "$DOWNLOAD_URL" -o "$TARGET_DIR/blsqui"
    sudo chmod +x "$TARGET_DIR/blsqui"
else
    curl -fsSL "$DOWNLOAD_URL" -o "$TARGET_DIR/blsqui"
    chmod +x "$TARGET_DIR/blsqui"
fi

echo "✅ Blsqui CLI installed successfully! Run 'blsqui' to get started."