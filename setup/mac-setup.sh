#!/bin/bash
set -e

echo "Установка k8s-jprof"

ZIP_URL="https://github.com/ZERDICORP/k8s-jprof/releases/download/latest/k8s-jprof-mac-release.zip"
TMP_ZIP="/tmp/k8s-jprof-mac-release.zip"
INSTALL_DIR="/usr/local/k8s-jprof"
BIN_SYMLINK="/usr/local/bin/k8s-jprof"

curl -L -o "$TMP_ZIP" "$ZIP_URL"

sudo mkdir -p "$INSTALL_DIR"
sudo unzip -o "$TMP_ZIP" -d "$INSTALL_DIR"
sudo chmod +x "$INSTALL_DIR/k8s-jprof"

[ -L "$BIN_SYMLINK" ] && sudo rm "$BIN_SYMLINK"
sudo ln -s "$INSTALL_DIR/k8s-jprof" "$BIN_SYMLINK"

rm "$TMP_ZIP"

echo "Установка завершена! Теперь можно запускать через терминал: k8s-jprof"
