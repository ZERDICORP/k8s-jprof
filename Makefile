.PHONY: bld_win64 bld_macos help prebuild_win prebuild_macos package_win package_macos

MEDIA_FILE=./media/logo_50.png

# Prebuild steps
prebuild_win:
	@mkdir -p ./bin/win64/media
	@cp $(MEDIA_FILE) ./bin/win64/media/

prebuild_macos:
	@mkdir -p ./bin/darwin/media
	@cp $(MEDIA_FILE) ./bin/darwin/media/

# Build targets
bld_win64: prebuild_win
	GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui" -o ./bin/win64/k8s-prf.exe ./src/.
	$(MAKE) package_win

bld_macos: prebuild_macos
	GOOS=darwin GOARCH=amd64 go build -o ./bin/darwin/k8s-prf ./src/.
	$(MAKE) package_macos

# Packaging
package_win:
	@cd ./bin/win64 && zip -r k8s-prf-win64.zip ./*

package_macos:
	@cd ./bin/darwin && zip -r k8s-prf-darwin.zip ./*

# Help
help:
	@echo "Available commands:"
	@echo "  bld_win64 - Build Windows GUI version and zip it"
	@echo "  bld_macos - Build macOS version and zip it"
