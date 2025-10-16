.PHONY: win mac win_release mac_release help prebuild_win prebuild_macos package_win package_macos package_win_release package_macos_release prebuild_win_release prebuild_macos_release

RSRC_DIR=resources

# ===========================
# WINDOWS BUILD (run on Windows)
# ===========================
win_syso:
	go-winres simply --icon media/icon.ico --manifest gui --admin
	cmd /c move /Y *.syso src\

prebuild_win: win_syso
	if not exist "bin\\win64" mkdir "bin\\win64"
	xcopy /E /I /Y "$(RSRC_DIR)\\*" "bin\\win64\\"

win: prebuild_win
	go build -ldflags="-H windowsgui" -o ./bin/win64/k8s-jprof.exe ./src/.
	powershell -Command "Get-ChildItem -Path 'src' -Filter '*.syso' | Remove-Item -Force"
	$(MAKE) package_win

package_win:
	powershell -Command "Compress-Archive -Path './bin/win64/*' -DestinationPath './bin/win64/k8s-jprof-win64.zip' -Force"

# ===========================
# WINDOWS RELEASE BUILD
# ===========================
prebuild_win_release: win_syso
	if not exist "bin\\win64_release" mkdir "bin\\win64_release"
	xcopy /E /I /Y "$(RSRC_DIR)\\*" "bin\\win64_release\\"

win_release: prebuild_win_release
	go build -ldflags="-H windowsgui" -o ./bin/win64_release/k8s-jprof.exe ./src/.
	powershell -Command "Get-ChildItem -Path 'src' -Filter '*.syso' | Remove-Item -Force"
	$(MAKE) package_win_release

package_win_release:
	powershell -Command "Compress-Archive -Path './bin/win64_release/*' -DestinationPath './bin/win64_release/k8s-jprof-win64-release.zip' -Force"

# ===========================
# MACOS BUILD (run on macOS)
# ===========================
prebuild_macos:
	@mkdir -p ./bin/darwin
	@cp -R ./$(RSRC_DIR)/* ./bin/darwin/

mac: prebuild_macos
	go build -o ./bin/darwin/k8s-jprof ./src/.
	$(MAKE) package_macos

package_macos:
	@cd ./bin/darwin && zip -r k8s-jprof-darwin.zip ./*

# ===========================
# MACOS RELEASE BUILD
# ===========================
prebuild_macos_release:
	@mkdir -p ./bin/darwin_release
	@cp -R ./$(RSRC_DIR)/* ./bin/darwin_release/

mac_release: prebuild_macos_release
	go build -o ./bin/darwin_release/k8s-jprof ./src/.
	$(MAKE) package_macos_release

package_macos_release:
	@cd ./bin/darwin_release && zip -r k8s-jprof-mac-release.zip ./*

# ===========================
# HELP
# ===========================
help:
	@echo "Available commands:"
	@echo "  win - Build Windows GUI version and zip it (run on Windows)"
	@echo "  mac - Build macOS version and zip it (run on macOS)"
	@echo "  win_release - Build Windows release version and zip it"
	@echo "  mac_release - Build macOS release version and zip it"
