#!/bin/bash
# build.sh: Cross-compiles the Go project for multiple platforms into ./build/

set -euo pipefail  # Exit on error, undefined vars, pipe failures

SRC_DIR="./src"
BIN_NAME="rpb_preprocessor"
BUILD_DIR="./build"

mkdir -p "$BUILD_DIR"

# Targets: GOOS_GOARCH
declare -A targets=(
  ["linux_arm64"]="linux/arm64"
  ["linux_amd64"]="linux/amd64"
  ["macos_arm64"]="darwin/arm64"
  ["macos_amd64"]="darwin/amd64"
  ["windows_amd64"]="windows/amd64"
)

echo "Building $BIN_NAME for ${!targets[*]}..."

for suffix in "${!targets[@]}"; do
  goos_arch="${targets[$suffix]}"
  IFS='/' read -r goos goarch <<< "$goos_arch"

  # Output filename: rpb_preprocessor_{suffix}.exe (add .exe only for Windows)
  output="$BUILD_DIR/$BIN_NAME"_"$suffix"
  if [[ "$goos" == "windows" ]]; then
    output+=".exe"
  fi

  echo "  Building $suffix -> $output"

  GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -o "$output" "$SRC_DIR"
done

echo "Build complete! Binaries in $BUILD_DIR/"
