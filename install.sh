#!/bin/bash

# Claude Auto-Deploy CLI Installation Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Installation directory (you can change this)
INSTALL_DIR="/usr/local/bin"

echo -e "${GREEN}Installing Claude Auto-Deploy CLI...${NC}"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${YELLOW}Warning: Go is not installed. Installing binary only...${NC}"

    # Check if binary exists
    if [ ! -f "bin/claude-auto" ]; then
        echo -e "${RED}Error: Binary not found. Please run 'make build' first.${NC}"
        exit 1
    fi

    # Copy binary to install directory
    echo "Installing binary to ${INSTALL_DIR}..."
    sudo cp bin/claude-auto ${INSTALL_DIR}/
    sudo chmod +x ${INSTALL_DIR}/claude-auto
else
    echo "Building from source..."
    make build

    echo "Installing binary to ${INSTALL_DIR}..."
    sudo cp bin/claude-auto ${INSTALL_DIR}/
    sudo chmod +x ${INSTALL_DIR}/claude-auto
fi

# Create config directory in user's home
CONFIG_DIR="$HOME/.claude-auto"
if [ ! -d "$CONFIG_DIR" ]; then
    echo "Creating configuration directory at ${CONFIG_DIR}..."
    mkdir -p ${CONFIG_DIR}
    cp configs/default.yaml ${CONFIG_DIR}/
    echo -e "${GREEN}Configuration file copied to ${CONFIG_DIR}/default.yaml${NC}"
fi

# Verify installation
if command -v claude-auto &> /dev/null; then
    echo -e "${GREEN}✅ Installation successful!${NC}"
    echo -e "${GREEN}Claude Auto-Deploy CLI version:${NC}"
    claude-auto --version
    echo ""
    echo -e "${GREEN}Usage:${NC}"
    echo "  claude-auto idea \"your project idea\""
    echo ""
    echo -e "${GREEN}Examples:${NC}"
    echo "  claude-auto idea \"create a todo app with React\""
    echo "  claude-auto idea \"build a REST API for blog\" --workers=5"
    echo "  claude-auto idea \"mobile app for fitness tracking\" --auto-approve"
else
    echo -e "${RED}❌ Installation failed. Please check the errors above.${NC}"
    exit 1
fi