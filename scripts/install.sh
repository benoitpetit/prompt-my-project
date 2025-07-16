#!/bin/bash

# Vérifier si Go est installé
check_go_installation() {
    if command -v go >/dev/null 2>&1; then
        echo "Go est installé. Voulez-vous utiliser 'go install' pour une installation simplifiée ? (o/n)"
        read -r answer
        if [[ "$answer" =~ ^[oO]$ ]]; then
            echo "Installation avec Go..."
            if go install github.com/benoitpetit/prompt-my-project@latest; then
                echo "✅ Installation réussie avec go install!"
                exit 0
            else
                echo "❌ Échec de l'installation avec go install. Tentative d'installation alternative..."
            fi
        else
            echo "Installation standard sélectionnée..."
        fi
    fi
}

# Dependencies check
command -v curl >/dev/null 2>&1 || { echo "Error: curl is required" >&2; exit 1; }
command -v tar >/dev/null 2>&1 || { echo "Error: tar is required" >&2; exit 1; }

# Check if Go is installed and offer go install method
check_go_installation

# Internet and GitHub connectivity check
echo "Checking GitHub connection..."
if ! curl -s --connect-timeout 5 --max-time 10 --head https://github.com > /dev/null; then
    echo "Error: No Internet connection or GitHub is unreachable"
    exit 1
fi

# OS and architecture detection
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    armv7l) ARCH="arm" ;;
esac

# Configuration
REPO="benoitpetit/prompt-my-project"
BINARY_NAME="pmp"
INSTALL_DIR="/usr/local/bin"
if [ "$OS" = "darwin" ]; then
    OS="mac"
fi

# Get the latest version from GitHub
echo "Retrieving the latest version..."
if ! LATEST_RELEASE=$(curl -s --connect-timeout 5 --max-time 15 "https://api.github.com/repos/$REPO/releases/latest"); then
    echo "Error: Unable to connect to GitHub API"
    exit 1
fi

if [ -z "$LATEST_RELEASE" ] || [[ "$LATEST_RELEASE" == *"Not Found"* ]]; then
    echo "Error: Unable to retrieve latest version information"
    exit 1
fi

# Version extraction correction
# The previous method doesn't work correctly with all API response formats
if command -v jq >/dev/null 2>&1; then
    # Use jq if available (more reliable)
    VERSION=$(echo "$LATEST_RELEASE" | jq -r .tag_name)
else
    # Fallback method with grep and sed
    VERSION=$(echo "$LATEST_RELEASE" | grep -o '"tag_name"[^,]*' | cut -d'"' -f4)
fi

if [ -z "$VERSION" ]; then
    # Try an alternative method if previous ones failed
    VERSION=$(echo "$LATEST_RELEASE" | grep -o '"tag_name": *"[^"]*"' | sed 's/.*: *"//;s/"$//')
    
    if [ -z "$VERSION" ]; then
        echo "Error: Unable to extract version number"
        echo "API Response (beginning): $(echo "$LATEST_RELEASE" | head -n 10)"
        exit 1
    fi
fi

# Display version for debugging
echo "Extracted version: $VERSION"

# Make sure the version is in the correct format for the download URL
# In build.sh, the version obtained by Git includes the "v" prefix if present in the tag
VERSION_WITHOUT_V=${VERSION#v}  # Remove 'v' at the beginning if present

# Check if the download URL exists
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/${BINARY_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
echo "Main URL: $DOWNLOAD_URL"

# Alternative without 'v' prefix in filename
ALT_DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/${BINARY_NAME}_${VERSION_WITHOUT_V}_${OS}_${ARCH}.tar.gz"
echo "Alternative URL: $ALT_DOWNLOAD_URL"

# Try the main URL first
if curl --output /dev/null --silent --head --fail "$DOWNLOAD_URL"; then
    echo "URL found: $DOWNLOAD_URL"
    FINAL_URL="$DOWNLOAD_URL"
elif curl --output /dev/null --silent --head --fail "$ALT_DOWNLOAD_URL"; then
    echo "Alternative URL found: $ALT_DOWNLOAD_URL"
    FINAL_URL="$ALT_DOWNLOAD_URL"
else
    echo "Error: The binary for your system ($OS/$ARCH) doesn't exist in version $VERSION"
    echo "URLs not found:"
    echo "- $DOWNLOAD_URL"
    echo "- $ALT_DOWNLOAD_URL"
    exit 1
fi

# Create a temporary directory
TMP_DIR=$(mktemp -d)
if [ $? -ne 0 ]; then
    echo "Error: Unable to create a temporary directory"
    exit 1
fi

cd $TMP_DIR || { echo "Error: Unable to access temporary directory"; exit 1; }

# Download and extract the archive
echo "Downloading from $FINAL_URL..."
if ! curl -L -o binary.tar.gz "$FINAL_URL"; then
    echo "Error during download"
    rm -rf $TMP_DIR
    exit 1
fi

if ! tar xzf binary.tar.gz; then
    echo "Error extracting the archive"
    rm -rf $TMP_DIR
    exit 1
fi

# Find the binary in the extracted archive (it might be in a subdirectory)
echo "Searching for binary..."
BINARY_PATH=$(find . -type f -name "$BINARY_NAME" | head -n 1)

if [ -z "$BINARY_PATH" ]; then
    echo "Error: Unable to find binary $BINARY_NAME in the archive"
    ls -la
    rm -rf $TMP_DIR
    exit 1
fi

echo "Binary found: $BINARY_PATH"

# Installation
echo "Installing to $INSTALL_DIR..."
if [ ! -w "$INSTALL_DIR" ]; then
    # Try with sudo if the user doesn't have write permissions to the install directory
    if command -v sudo >/dev/null 2>&1; then
        sudo mv "$BINARY_PATH" $INSTALL_DIR/$BINARY_NAME || { echo "Error during installation"; exit 1; }
        sudo chmod +x $INSTALL_DIR/$BINARY_NAME || { echo "Error changing permissions"; exit 1; }
    else
        echo "Error: Insufficient permissions to install in $INSTALL_DIR and sudo is not available"
        exit 1
    fi
else
    mv "$BINARY_PATH" $INSTALL_DIR/$BINARY_NAME || { echo "Error during installation"; exit 1; }
    chmod +x $INSTALL_DIR/$BINARY_NAME || { echo "Error changing permissions"; exit 1; }
fi

# Cleanup
cd - > /dev/null
rm -rf $TMP_DIR

echo "✅ Installation complete! You can use 'pmp --help' to see available options."

# Installer l'autocomplétion Bash si possible
if command -v pmp >/dev/null 2>&1; then
    if [ -d /etc/bash_completion.d ]; then
        pmp completion bash | sudo tee /etc/bash_completion.d/pmp > /dev/null
        echo "✅ Bash completion installed in /etc/bash_completion.d/pmp"
    elif [ -d /usr/local/etc/bash_completion.d ]; then
        pmp completion bash | sudo tee /usr/local/etc/bash_completion.d/pmp > /dev/null
        echo "✅ Bash completion installed in /usr/local/etc/bash_completion.d/pmp"
    else
        echo "⚠️  Could not find bash_completion.d directory. Please install manually:"
        echo "    pmp completion bash > /etc/bash_completion.d/pmp"
    fi
    # Zsh (si fpath existe)
    if [ -n "$ZSH_VERSION" ] && [ -n "${fpath[1]}" ]; then
        pmp completion zsh > "${fpath[1]}/_pmp"
        echo "✅ Zsh completion installed in ${fpath[1]}/_pmp"
    fi
fi

# Ajoute la complétion pour ./pmp si le script est sourcé dans le dossier courant
if [ -f /etc/bash_completion.d/pmp ]; then
    echo 'complete -o default -F __start_pmp ./pmp' | sudo tee -a /etc/bash_completion.d/pmp > /dev/null
elif [ -f /usr/local/etc/bash_completion.d/pmp ]; then
    echo 'complete -o default -F __start_pmp ./pmp' | sudo tee -a /usr/local/etc/bash_completion.d/pmp > /dev/null
fi
