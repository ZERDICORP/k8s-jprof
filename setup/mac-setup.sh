#!/bin/bash
set -e

echo "Установка k8s-jprof"

ZIP_URL="https://github.com/ZERDICORP/k8s-jprof/releases/download/1.0.beta/k8s-jprof-mac-release.zip"
TMP_ZIP="/tmp/k8s-jprof-mac-release.zip"
INSTALL_DIR="/usr/local/k8s-jprof"
BIN_WRAPPER="/usr/local/bin/k8s-jprof"

curl -L -o "$TMP_ZIP" "$ZIP_URL"

sudo mkdir -p "$INSTALL_DIR"
sudo unzip -o "$TMP_ZIP" -d "$INSTALL_DIR"
sudo chmod +x "$INSTALL_DIR/k8s-jprof"

sudo tee "$BIN_WRAPPER" > /dev/null <<EOF
#!/bin/bash
"$INSTALL_DIR/k8s-jprof" "\$@"
EOF

sudo chmod +x "$BIN_WRAPPER"

# Удаляем временный файл
rm "$TMP_ZIP"

echo "Установка завершена! Теперь можно запускать через терминал: k8s-jprof"
