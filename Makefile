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
# Local signing identity for install (auto-detect Developer ID cert hash from Keychain, no Apple notarization)
LOCAL_SIGN_IDENTITY  ?= $(shell security find-identity -v -p codesigning 2>/dev/null | awk '/Developer ID Application:/{print $$2; exit}')

# 
.PHONY: help dev build clean install check-deps update-version cli

# Default target
help:
	@echo "Available targets:"
	@echo "  dev      - Start the development server"
	@echo "  build    - Build the application"
# 
	@echo "  clean    - Clean build artifacts"
	@echo "  install  - Install dependencies"
	@echo "  check-deps - Check if required tools are installed"
	@echo "  update-version - Update version across all sources (make update-version 0.7.0)"
	@echo "  cli      - Symlink the installed app as 'prochub' into PATH for CLI usage"

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

# 

# Build the application with DevTools enabled (F12 opens inspector)
# Note: macOS builds use private WebKit APIs - not suitable for App Store submission
build-devtools: check-deps
	wails build -tags devtools

# 

# Update version across all sources (app.go, package.json, package-lock.json, screenshot mock)
# Usage: make update-version 0.7.0
# The version is passed as a positional argument, so declare it as a no-op target.
ifneq ($(word 2,$(MAKECMDGOALS)),)
$(eval $(word 2,$(MAKECMDGOALS)): ; @true)
endif

update-version:
	@V="$(word 2,$(MAKECMDGOALS))"; \
	test -n "$$V" || { echo "Usage: make update-version 0.7.0"; exit 1; }; \
	bash scripts/update-version.sh "$$V"

# Clean build artifacts
clean:
	rm -rf build/bin
	rm -rf frontend/dist
	rm -rf frontend/node_modules
	go clean

# 

# 

# 
