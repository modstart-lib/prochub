# Makefile for ProcHub Wails Application

# App Store / TestFlight signing config
BUNDLE_ID            = com.modstart.prochub
APP_NAME             = ProcHub
APP_PATH             = build/bin/$(APP_NAME).app
PROVISION_PROFILE   ?= $(HOME)/Library/MobileDevice/Provisioning\ Profiles/$(APP_NAME)_AppStore.provisionprofile
# Application signing identity: "3rd Party Mac Developer Application: Your Name (TEAMID)"
SIGN_IDENTITY        ?= "3rd Party Mac Developer Application"
# Installer signing identity: "3rd Party Mac Developer Installer: Your Name (TEAMID)"
INSTALLER_IDENTITY   ?= "3rd Party Mac Developer Installer"

.PHONY: help dev build clean install check-deps

# Default target
help:
	@echo "Available targets:"
	@echo "  dev      - Start the development server"
	@echo "  build    - Build the application"
	@echo "  clean    - Clean build artifacts"
	@echo "  install  - Install dependencies"
	@echo "  check-deps - Check if required tools are installed"

# Check if required tools are installed
check-deps:
	@command -v go >/dev/null 2>&1 || { echo "Go is not installed. Please install Go."; exit 1; }
	@command -v wails >/dev/null 2>&1 || { echo "Wails is not installed. Please install Wails: go install github.com/wailsapp/wails/v2/cmd/wails@latest"; exit 1; }
	@command -v npm >/dev/null 2>&1 || { echo "npm is not installed. Please install Node.js and npm."; exit 1; }

# Install dependencies
install: check-deps
	cd frontend && npm install
	go mod tidy

# Start development server
dev: check-deps
	wails dev

# Build the application
build: check-deps
	wails build

# Clean build artifacts
clean:
	rm -rf build/bin
	rm -rf frontend/dist
	rm -rf frontend/node_modules
	go clean

build_and_install:
	$(MAKE) install
	$(MAKE) build
	sudo rm -rfv /Applications/ProcHub.app
	sudo cp -rv build/bin/ProcHub.app /Applications/ProcHub.app
