.PHONY: win mac help prebuild_win prebuild_macos package_win package_macos

MEDIA_DIR=media

# ===========================
# WINDOWS BUILD (run on Windows)
# ===========================
prebuild_win:
	if not exist "bin\\win64\\media" mkdir "bin\\win64\\media"
	xcopy /E /I /Y "$(MEDIA_DIR)" "bin\\win64\\media\\"

win: prebuild_win
	go build -ldflags="-H windowsgui" -o ./bin/win64/k8s-prf.exe ./src/.
	$(MAKE) package_win

package_win:
	powershell -Command "Compress-Archive -Path './bin/win64/*' -DestinationPath './bin/win64/k8s-prf-win64.zip' -Force"

# ===========================
# MACOS BUILD (run on macOS)
# ===========================
prebuild_macos:
	@mkdir -p ./bin/darwin/media
	@cp -R ./media/* ./bin/darwin/media/

mac: prebuild_macos
	go build -o ./bin/darwin/k8s-prf ./src/.
	$(MAKE) package_macos

package_macos:
	@cd ./bin/darwin && zip -r k8s-prf-darwin.zip ./*

# ===========================
# HELP
# ===========================
help:
	@echo "Available commands:"
	@echo "  win - Build Windows GUI version and zip it (run on Windows)"
	@echo "  mac - Build macOS version and zip it (run on macOS)"
