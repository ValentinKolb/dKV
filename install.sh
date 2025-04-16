#!/bin/bash

# Print banner
echo "====================================="
echo "dKV Installation Script"
echo "====================================="
echo "This script will install dKV from https://github.com/ValentinKolb/dKV"
echo

# Check if Git is installed
if ! command -v git &> /dev/null; then
    echo "Error: Git is not installed. Please install Git before continuing."
    exit 1
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go before continuing."
    exit 1
fi

# Create temporary directory
TEMP_DIR=$(mktemp -d)
echo "Created temporary directory: $TEMP_DIR"

# Clean up on exit
cleanup() {
    echo "Cleaning up temporary directory..."
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

# Clone the repository
echo "Cloning repository from https://github.com/ValentinKolb/dKV..."
if ! git clone https://github.com/ValentinKolb/dKV "$TEMP_DIR"; then
    echo "Error: Failed to clone repository."
    exit 1
fi

# Change to the cloned directory
cd "$TEMP_DIR" || exit 1

# Target directory for installation
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    INSTALL_DIR="/usr/local/bin"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Linux
    INSTALL_DIR="/usr/local/bin"
else
    echo "Error: Operating system not supported."
    exit 1
fi

# Check if target directory exists and is writable
if [ ! -d "$INSTALL_DIR" ] || [ ! -w "$INSTALL_DIR" ]; then
    echo "Warning: The target directory $INSTALL_DIR does not exist or is not writable."
    echo "We'll need to use sudo for installation."
    SUDO="sudo"
else
    SUDO=""
fi

echo "Compiling the Go program..."
# Compile Go program
if ! go build -o dkv; then
    echo "Error: Failed to compile the Go program."
    exit 1
fi

echo "Installing dKV to $INSTALL_DIR..."
# Move the executable to the target directory
if ! $SUDO mv dkv "$INSTALL_DIR/"; then
    echo "Error: Failed to move the executable to $INSTALL_DIR."
    exit 1
fi

# Make the file executable
if ! $SUDO chmod +x "$INSTALL_DIR/dkv"; then
    echo "Error: Failed to make the file executable."
    exit 1
fi

echo "Installation complete! You can now run 'dkv' in the terminal."