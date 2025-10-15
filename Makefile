.PHONY: bld_win64

bld_win64:
	go build -ldflags="-H windowsgui" -o ./bin/k8s-prf.exe ./src/.

# Help
help:
	@echo "Available commands:"
	@echo "  bld_win64 - Build Windows GUI version"