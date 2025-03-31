#!/bin/bash

# Configuration
APP_NAME="pmp"
BINARY_NAME=${APP_NAME}
DIST_DIR="dist"

# Déterminer la version suggérée depuis git ou utiliser la version par défaut
SUGGESTED_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v1.0.2")

# Demander à l'utilisateur quelle version builder
echo -n "Version à builder [$SUGGESTED_VERSION]: "
read USER_VERSION
VERSION=${USER_VERSION:-$SUGGESTED_VERSION}  # Utiliser la saisie utilisateur ou la version suggérée si vide
echo "Building version: $VERSION"

# Note: Cette version peut inclure un préfixe 'v', qui sera inclus dans le nom du fichier
# Les scripts d'installation sont configurés pour gérer les deux formats (avec ou sans 'v')

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

        # Créer le nom de l'archive et le dossier temporaire
        archive_name="${APP_NAME}_${VERSION}_${platform}_${arch}"
        tmp_dir="$DIST_DIR/$archive_name"
        mkdir -p "$tmp_dir"

        # Compiler directement dans le dossier temporaire
        go build -ldflags="-s -w" -o "$tmp_dir/$output_name"

        # Copier les fichiers de documentation
        cp README.md LICENSE "$tmp_dir/" 2>/dev/null || true

        # Créer l'archive
        pushd $DIST_DIR > /dev/null
        if [ $platform = "windows" ]; then
            # Pour Windows, utiliser zip
            pushd "$archive_name" > /dev/null
            zip -r "../${archive_name}.zip" ./*
            popd > /dev/null
        else
            # Pour Linux et macOS, utiliser tar
            tar -czf "${archive_name}.tar.gz" -C "$archive_name" .
        fi
        popd > /dev/null

        # Nettoyer le dossier temporaire
        rm -rf "$DIST_DIR/$archive_name"
    done
done

# Générer les checksums
echo "Génération des checksums..."
pushd $DIST_DIR > /dev/null
echo "# Checksums SHA-256" > checksums.txt
for file in *.tar.gz *.zip; do
    if [ -f "$file" ]; then
        sha256sum "$file" >> checksums.txt
    fi
done
popd > /dev/null

echo "Build terminé ! Les binaires, archives et checksums sont disponibles dans le dossier $DIST_DIR"
