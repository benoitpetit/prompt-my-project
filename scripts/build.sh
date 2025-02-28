#!/bin/bash

# Configuration
APP_NAME="pmp"
BINARY_NAME=${APP_NAME}
DIST_DIR="dist"

# Obtenir la version depuis git ou utiliser la version par défaut
VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v1.0.0")

# Créer le dossier de distribution
rm -rf $DIST_DIR
mkdir -p $DIST_DIR

# Plateformes et architectures à builder
PLATFORMS=("linux" "darwin" "windows")
ARCHITECTURES=("amd64" "arm64")

# Construction pour chaque plateforme et architecture
for platform in "${PLATFORMS[@]}"; do
    for arch in "${ARCHITECTURES[@]}"; do
        output_name=$BINARY_NAME
        if [ $platform = "windows" ]; then
            output_name+='.exe'
        fi

        echo "Building for $platform/$arch..."
        export GOOS=$platform
        export GOARCH=$arch

        # Créer le nom de l'archive
        archive_name="${APP_NAME}_${VERSION}_${platform}_${arch}"
        mkdir -p "$DIST_DIR/$archive_name"

        # Compiler
        go build -ldflags="-s -w" -o "$DIST_DIR/$archive_name/$output_name"

        # Copier les fichiers de documentation
        cp README.md LICENSE "$DIST_DIR/$archive_name/" 2>/dev/null || true

        # Créer l'archive
        cd $DIST_DIR
        if [ $platform = "windows" ]; then
            zip -r "${archive_name}.zip" "$archive_name"
        else
            tar czf "${archive_name}.tar.gz" "$archive_name"
        fi
        cd ..

        # Nettoyer le dossier temporaire
        rm -rf "$DIST_DIR/$archive_name"
    done
done

# Générer les checksums
echo "Génération des checksums..."
cd $DIST_DIR
echo "# Checksums SHA-256" > checksums.txt
for file in *.tar.gz *.zip; do
    if [ -f "$file" ]; then
        sha256sum "$file" >> checksums.txt
    fi
done
cd ..

echo "Build terminé ! Les binaires, archives et checksums sont disponibles dans le dossier $DIST_DIR"
