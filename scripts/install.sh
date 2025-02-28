#!/bin/bash

# Vérification des dépendances
command -v curl >/dev/null 2>&1 || { echo "Erreur: curl est requis" >&2; exit 1; }
command -v tar >/dev/null 2>&1 || { echo "Erreur: tar est requis" >&2; exit 1; }

# Vérification de la connexion Internet
curl -s --head  --request GET "https://api.github.com" | grep "200 OK" > /dev/null
if [ $? -ne 0 ]; then
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
VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | cut -d'"' -f4)

if [ -z "$VERSION" ]; then
    echo "Erreur: Impossible de récupérer la dernière version"
    exit 1
fi

# Construire l'URL de téléchargement
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/${BINARY_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"

# Créer un dossier temporaire
TMP_DIR=$(mktemp -d)
cd $TMP_DIR

# Télécharger et extraire l'archive
echo "Téléchargement de $DOWNLOAD_URL..."
curl -L -o binary.tar.gz "$DOWNLOAD_URL"
tar xzf binary.tar.gz

# Installation
echo "Installation dans $INSTALL_DIR..."
sudo mv $BINARY_NAME $INSTALL_DIR/
sudo chmod +x $INSTALL_DIR/$BINARY_NAME

# Nettoyage
cd ..
rm -rf $TMP_DIR

echo "Installation terminée ! Vous pouvez utiliser 'pmp --help' pour voir les options disponibles."
