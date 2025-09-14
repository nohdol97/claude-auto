#!/bin/bash

# Claude Auto-Deploy CLI Installation Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Installation directory (you can change this)
INSTALL_DIR="/usr/local/bin"

echo -e "${GREEN}Installing Claude Auto-Deploy CLI...${NC}"

# Function to install Go using Homebrew
install_go_with_brew() {
    echo -e "${BLUE}Installing Go using Homebrew...${NC}"
    brew install go
    echo -e "${GREEN}Go installed successfully!${NC}"
}

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${YELLOW}Go is not installed.${NC}"

    # Check if Homebrew is available
    if command -v brew &> /dev/null; then
        echo -e "${BLUE}Homebrew is available. Would you like to install Go? (y/n)${NC}"
        read -r response
        if [[ "$response" == "y" || "$response" == "Y" ]]; then
            install_go_with_brew
            # Reload PATH
            export PATH="/usr/local/go/bin:$PATH"
            export PATH="$HOME/go/bin:$PATH"
        else
            echo -e "${YELLOW}Attempting to download pre-built binary...${NC}"

            # Try to download pre-built binary (if available)
            # For now, we'll exit and ask user to install Go
            echo -e "${RED}Please install Go first:${NC}"
            echo -e "  Option 1: ${BLUE}brew install go${NC}"
            echo -e "  Option 2: Download from ${BLUE}https://golang.org/dl/${NC}"
            echo ""
            echo -e "After installing Go, run this script again."
            exit 1
        fi
    else
        echo -e "${RED}Go is required to build claude-auto.${NC}"
        echo -e "Please install Go from: ${BLUE}https://golang.org/dl/${NC}"
        echo -e "Or install Homebrew first: ${BLUE}/bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"${NC}"
        exit 1
    fi
fi

# Now Go should be installed, let's build
echo "Building from source..."
go mod download
go build -o bin/claude-auto cmd/claude-auto/main.go

if [ ! -f "bin/claude-auto" ]; then
    echo -e "${RED}Build failed. Please check the error messages above.${NC}"
    exit 1
fi

echo "Installing binary to ${INSTALL_DIR}..."
sudo cp bin/claude-auto ${INSTALL_DIR}/
sudo chmod +x ${INSTALL_DIR}/claude-auto

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