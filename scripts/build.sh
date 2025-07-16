#!/bin/bash

# Configuration
APP_NAME="pmp"
BINARY_NAME=${APP_NAME}
DIST_DIR="dist"

# Get suggested version from git or use default
SUGGESTED_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v1.0.2")

# Version management: argument or user input
if [ -n "$1" ]; then
    VERSION="$1"
    echo "Building version: $VERSION"
else
    echo -n "Version to build [$SUGGESTED_VERSION]: "
    read USER_VERSION
    VERSION=${USER_VERSION:-$SUGGESTED_VERSION}
    echo "Building version: $VERSION"
fi

# Remove and recreate dist directory
rm -rf $DIST_DIR
mkdir -p $DIST_DIR

# Platforms and architectures to build
PLATFORMS=("linux" "darwin" "windows")
ARCHITECTURES=("amd64" "arm64")

# Build for each platform and architecture
for platform in "${PLATFORMS[@]}"; do
    for arch in "${ARCHITECTURES[@]}"; do
        output_name=$BINARY_NAME
        if [ $platform = "windows" ]; then
            output_name+='.exe'
        fi

        export GOOS=$platform
        export GOARCH=$arch

        archive_name="${APP_NAME}_${VERSION}_${platform}_${arch}"
        tmp_dir="$DIST_DIR/$archive_name"
        mkdir -p "$tmp_dir"

        go build -ldflags="-s -w" -o "$tmp_dir/$output_name"
        cp README.md LICENSE "$tmp_dir/" 2>/dev/null || true

        pushd $DIST_DIR > /dev/null
        if [ $platform = "windows" ]; then
            pushd "$archive_name" > /dev/null
            zip -rq "../${archive_name}.zip" ./*
            popd > /dev/null
        else
            tar -czf "${archive_name}.tar.gz" -C "$archive_name" .
        fi
        popd > /dev/null

        rm -rf "$DIST_DIR/$archive_name"
    done
done

# Generate checksums
pushd $DIST_DIR > /dev/null
echo "# SHA-256 Checksums" > checksums.txt
for file in *.tar.gz *.zip; do
    if [ -f "$file" ]; then
        sha256sum "$file" >> checksums.txt
    fi
done
popd > /dev/null

echo "Build complete. Artifacts are in $DIST_DIR."
