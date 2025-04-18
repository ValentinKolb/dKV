#!/bin/bash

# Default install directory
INSTALL_DIR="/usr/local/bin"
FROM_SOURCE=false

# Help function
show_help() {
    echo "dKV installation script, version 0.1.0"
    echo
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Options:"
    echo "  -h, --help     Show this help message and exit"
    echo "  --source       Install from source code instead of using pre-compiled binaries"
    echo "  --path=DIR     Set custom installation directory (default: /usr/local/bin)"
    echo
    echo "Examples:"
    echo "  $0                   # Install binary to default location"
    echo "  $0 --source          # Install from source"
    echo "  $0 --path=/opt/bin   # Install to custom location"
    echo "  $0 --source --path=/home/user/bin   # Install from source to custom location"
}

# Process command line arguments
for arg in "$@"; do
    case $arg in
        -h|--help)
            show_help
            exit 0
            ;;
        --source)
            FROM_SOURCE=true
            shift
            ;;
        --path=*)
            INSTALL_DIR="${arg#*=}"
            shift
            ;;
    esac
done

# Print banner
echo "========================"
echo "dKV Installation Script"
echo "========================"
echo

# Determine OS and architecture
OS=$(uname -s)
ARCH=$(uname -m)

# Map architecture names
if [ "$ARCH" = "x86_64" ]; then
    ARCH_NAME="x86_64"
elif [[ "$ARCH" = "arm64" || "$ARCH" = "aarch64" ]]; then
    ARCH_NAME="arm64"
else
    echo "Error: Unsupported architecture: $ARCH"
    exit 1
fi

# Install from source function
install_from_source() {
    echo "Installing dKV from source..."

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
    echo "Cloning repository from https://github.com/ValentinKolb/dKV ..."
    if ! git clone https://github.com/ValentinKolb/dKV "$TEMP_DIR"; then
        echo "Error: Failed to clone repository."
        exit 1
    fi

    # Change to the cloned directory
    cd "$TEMP_DIR" || exit 1

    echo "Compiling the Go program..."
    # Compile Go program
    if ! go build -o dkv; then
        echo "Error: Failed to compile the Go program."
        exit 1
    fi

    # Move the executable to target directory
    install_binary "./dkv"
}

# Install from binary function
install_from_binary() {
    echo "Installing dKV binary for $OS $ARCH_NAME..."

    # Create temporary directory
    TEMP_DIR=$(mktemp -d)
    echo "Created temporary directory: $TEMP_DIR"

    # Clean up on exit
    cleanup() {
        echo "Cleaning up temporary directory..."
        rm -rf "$TEMP_DIR"
    }
    trap cleanup EXIT

    # Set download URL based on OS and architecture
    if [ "$OS" = "Darwin" ]; then
        # macOS
        DOWNLOAD_URL="https://github.com/valentinkolb/dkv/releases/latest/download/dkv_Darwin_${ARCH_NAME}.zip"
        ARCHIVE_TYPE="zip"
    elif [ "$OS" = "Linux" ]; then
        # Linux
        DOWNLOAD_URL="https://github.com/valentinkolb/dkv/releases/latest/download/dkv_Linux_${ARCH_NAME}.tar.gz"
        ARCHIVE_TYPE="tar.gz"
    else
        echo "Error: Unsupported operating system: $OS"
        exit 1
    fi

    echo "Downloading from: $DOWNLOAD_URL"

    # Download the binary
    cd "$TEMP_DIR" || exit 1
    if [ "$ARCHIVE_TYPE" = "zip" ]; then
        if ! curl -L "$DOWNLOAD_URL" -o dkv.zip; then
            echo "Error: Failed to download the binary."
            exit 1
        fi
        # Extract the zip file
        if ! unzip dkv.zip; then
            echo "Error: Failed to extract the archive."
            exit 1
        fi
    else
        if ! curl -L "$DOWNLOAD_URL" -o dkv.tar.gz; then
            echo "Error: Failed to download the binary."
            exit 1
        fi
        # Extract the tar.gz file
        if ! tar -xzf dkv.tar.gz; then
            echo "Error: Failed to extract the archive."
            exit 1
        fi
    fi

    # Find the binary
    if [ -f "./dkv" ]; then
        install_binary "./dkv"
    else
        echo "Error: Could not find the dkv binary in the extracted files."
        exit 1
    fi
}

# Function to install binary to target directory
install_binary() {
    BINARY_PATH=$1

    echo "Installing dKV to $INSTALL_DIR..."

    # Create directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        if ! mkdir -p "$INSTALL_DIR"; then
            echo "Error: Failed to create directory $INSTALL_DIR. Trying with sudo..."
            if ! sudo mkdir -p "$INSTALL_DIR"; then
                echo "Error: Failed to create directory even with sudo."
                exit 1
            fi
        fi
    fi

    # Check if target directory is writable
    if [ ! -w "$INSTALL_DIR" ]; then
        echo "The target directory $INSTALL_DIR is not writable. Using sudo for installation."
        SUDO="sudo"
    else
        SUDO=""
    fi

    # Move the executable to the target directory
    if ! $SUDO mv "$BINARY_PATH" "$INSTALL_DIR/dkv"; then
        echo "Error: Failed to move the executable to $INSTALL_DIR."
        exit 1
    fi

    # Make the file executable
    if ! $SUDO chmod +x "$INSTALL_DIR/dkv"; then
        echo "Error: Failed to make the file executable."
        exit 1
    fi

    echo "Installation complete! You can now run 'dkv' from the terminal."
    echo "Installed at: $INSTALL_DIR/dkv"

    # Check if the directory is in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo "Note: $INSTALL_DIR is not in your PATH."
        echo "You may need to add it to your PATH or use the full path to run dkv."
    fi
}

# Main installation process
if [ "$FROM_SOURCE" = true ]; then
    install_from_source
else
    install_from_binary
fi