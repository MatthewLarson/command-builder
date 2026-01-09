BINARY_NAME=cb
BUILD_DIR=bin

.PHONY: all build clean install package

all: build

build:
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) cmd/cb/main.go

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) dist

install:
	@echo "Installing binary..."
	go install cmd/cb/main.go
	@echo ""
	@echo "IMPORTANT: To enable history integration, add the following to your ~/.bashrc or ~/.zshrc:"
	@echo "  source $(PWD)/scripts/cb.bash"

package:
	@echo "Packaging for Debian..."
	@mkdir -p dist/usr/bin
	@mkdir -p dist/usr/share/command-builder
	@mkdir -p dist/DEBIAN
	@cp packaging/debian/control dist/DEBIAN/
	@cp scripts/cb.bash dist/usr/share/command-builder/
	@go build -o dist/usr/bin/$(BINARY_NAME) cmd/cb/main.go
	@chmod 755 dist/usr/bin/$(BINARY_NAME)
	@dpkg-deb --build dist command-builder.deb
	@echo "Package created: command-builder.deb"
