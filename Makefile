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
.PHONY: help dev build clean install check-deps build-and-install

# Default target
help:
	@echo "Available targets:"
	@echo "  dev      - Start the development server"
	@echo "  build    - Build the application"
	@echo "  build-and-install - Build, sign locally (Developer ID, no notarization), install to /Applications"
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

# 

# Build the application with DevTools enabled (F12 opens inspector)
# Note: macOS builds use private WebKit APIs - not suitable for App Store submission
build-devtools: check-deps
	wails build -tags devtools

# 

# Clean build artifacts
clean:
	rm -rf build/bin
	rm -rf frontend/dist
	rm -rf frontend/node_modules
	go clean

# Build the app, sign it locally (Developer ID cert if available, otherwise ad-hoc),
# then install it into /Applications and launch. No Apple notarization is performed.
build-and-install: check-deps
	$(MAKE) install
	$(MAKE) build
	@if [ -z "$(LOCAL_SIGN_IDENTITY)" ]; then \
		echo ">>> No Developer ID certificate found, falling back to ad-hoc signing"; \
		codesign --force --deep --sign - $(APP_PATH); \
	else \
		echo ">>> Signing $(APP_PATH) with '$(LOCAL_SIGN_IDENTITY)' (local only, no notarization)"; \
		codesign --force --deep --sign "$(LOCAL_SIGN_IDENTITY)" $(APP_PATH); \
	fi
	codesign --verify --deep --strict --verbose=2 $(APP_PATH)
	@echo ">>> Stopping running ProcHub instances"
	pkill -f '/ProcHub.app' 2>/dev/null || true
	@echo ">>> Installing to /Applications/ProcHub.app"
	@rm -rf /Applications/ProcHub.app; \
	if cp -R $(APP_PATH) /Applications/ProcHub.app; then \
		echo ">>> Installed to /Applications/ProcHub.app (no sudo needed)"; \
	else \
		echo ">>> /Applications not writable by current user, retrying with sudo..."; \
		sudo rm -rf /Applications/ProcHub.app; \
		sudo cp -R $(APP_PATH) /Applications/ProcHub.app; \
	fi
	codesign --verify --deep --strict --verbose=2 /Applications/ProcHub.app
	@echo ">>> Done. Launching ProcHub..."
	open /Applications/ProcHub.app

# 
