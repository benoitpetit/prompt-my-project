#!/bin/bash

# Vérification des dépendances
command -v curl >/dev/null 2>&1 || { echo "Erreur: curl est requis" >&2; exit 1; }
command -v tar >/dev/null 2>&1 || { echo "Erreur: tar est requis" >&2; exit 1; }

# Vérification de la connexion Internet et GitHub
echo "Vérification de la connexion à GitHub..."
if ! curl -s --connect-timeout 5 --max-time 10 --head https://github.com > /dev/null; then
    echo "Erreur: Pas de connexion Internet ou GitHub inaccessible"
    exit 1
fi

# Détection de l'OS et de l'architecture
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

# Obtenir la dernière version depuis GitHub
echo "Récupération de la dernière version..."
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest")
if [ -z "$LATEST_RELEASE" ]; then
    echo "Erreur: Impossible de récupérer les informations de la dernière version"
    exit 1
fi

VERSION=$(echo "$LATEST_RELEASE" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$VERSION" ]; then
    echo "Erreur: Impossible d'extraire le numéro de version"
    exit 1
fi

# Construire l'URL de téléchargement
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/${BINARY_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
echo "Version détectée: $VERSION"
echo "Architecture: $OS/$ARCH"

# Créer un dossier temporaire
TMP_DIR=$(mktemp -d)
if [ $? -ne 0 ]; then
    echo "Erreur: Impossible de créer un dossier temporaire"
    exit 1
fi

cd $TMP_DIR || { echo "Erreur: Impossible d'accéder au dossier temporaire"; exit 1; }

# Télécharger et extraire l'archive
echo "Téléchargement depuis $DOWNLOAD_URL..."
if ! curl -L -o binary.tar.gz "$DOWNLOAD_URL"; then
    echo "Erreur lors du téléchargement"
    rm -rf $TMP_DIR
    exit 1
fi

if ! tar xzf binary.tar.gz; then
    echo "Erreur lors de l'extraction de l'archive"
    rm -rf $TMP_DIR
    exit 1
fi

# Installation
echo "Installation dans $INSTALL_DIR..."
if [ ! -w "$INSTALL_DIR" ]; then
    # Essayer avec sudo si le dossier n'est pas accessible en écriture
    if command -v sudo >/dev/null 2>&1; then
        sudo mv $BINARY_NAME $INSTALL_DIR/ || { echo "Erreur lors de l'installation"; exit 1; }
        sudo chmod +x $INSTALL_DIR/$BINARY_NAME || { echo "Erreur lors de la modification des permissions"; exit 1; }
    else
        echo "Erreur: Permissions insuffisantes pour installer dans $INSTALL_DIR et sudo n'est pas disponible"
        exit 1
    fi
else
    mv $BINARY_NAME $INSTALL_DIR/ || { echo "Erreur lors de l'installation"; exit 1; }
    chmod +x $INSTALL_DIR/$BINARY_NAME || { echo "Erreur lors de la modification des permissions"; exit 1; }
fi

# Nettoyage
cd - > /dev/null
rm -rf $TMP_DIR

echo "✅ Installation terminée ! Vous pouvez utiliser 'pmp --help' pour voir les options disponibles."
