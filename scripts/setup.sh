#!/bin/bash
# Development setup script for jobsearch-mcp

set -e

echo "Setting up jobsearch-mcp development environment..."

# Check Go version
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go 1.22+"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "Go version: $GO_VERSION"

# Check Python version
if ! command -v python3 &> /dev/null; then
    echo "Error: Python 3 is not installed. Please install Python 3.11+"
    exit 1
fi

PYTHON_VERSION=$(python3 --version | awk '{print $2}')
echo "Python version: $PYTHON_VERSION"

# Check Ollama
if ! command -v ollama &> /dev/null; then
    echo "Warning: Ollama is not installed. Install from https://ollama.ai"
else
    echo "Ollama: installed"
    if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
        echo "Ollama: running"
    else
        echo "Warning: Ollama is not running. Start with 'ollama serve'"
    fi
fi

# Install Go dependencies
echo ""
echo "Installing Go dependencies..."
go mod tidy

# Install Python dependencies
echo ""
echo "Installing Python dependencies..."
cd classifier
python3 -m pip install -e ".[dev]" --quiet
cd ..

# Create config directory
CONFIG_DIR="$HOME/.config/jobsearch"
DATA_DIR="$HOME/.local/share/jobsearch"

mkdir -p "$CONFIG_DIR"
mkdir -p "$DATA_DIR"

echo ""
echo "Setup complete!"
echo ""
echo "Next steps:"
echo "  1. Set up Gmail API credentials (see README.md)"
echo "  2. Save credentials.json to $CONFIG_DIR/"
echo "  3. Run 'jobsearch config init' to create config file"
echo "  4. Run 'jobsearch sync' to authenticate and fetch emails"
