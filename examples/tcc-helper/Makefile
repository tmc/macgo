# Makefile for tcc-helper with code signing support
#
# Usage:
#   make              - Build with automatic signing (Developer ID if available, stable ad-hoc otherwise)
#   make dev          - Build for development with debug output
#   make clean        - Clean build artifacts
#   make install      - Install to ~/bin
#   make help         - Show this help

.PHONY: all dev clean install help

BINARY_NAME := tcc-helper
GO := go
INSTALL_DIR := ~/bin

# Default target
all: build

# Build with automatic signing
build:
	@echo "Building $(BINARY_NAME) with automatic code signing..."
	$(GO) build -o $(BINARY_NAME)
	@echo ""
	@echo "Build complete: $(BINARY_NAME)"
	@echo ""
	@echo "The binary will use:"
	@echo "  - Developer ID Application certificate (if available)"
	@echo "  - Stable ad-hoc signature (if no Developer ID)"
	@echo ""
	@echo "This ensures permissions persist across rebuilds."

# Build for development with debug output
dev:
	@echo "Building $(BINARY_NAME) in development mode..."
	MACGO_DEBUG=1 $(GO) build -o $(BINARY_NAME)
	@echo ""
	@echo "Build complete with debug logging enabled."
	@echo "Run with MACGO_DEBUG=1 to see detailed signing information."

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	@echo "Clean complete."

# Install to ~/bin
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY_NAME) $(INSTALL_DIR)/
	@echo ""
	@echo "Installation complete: $(INSTALL_DIR)/$(BINARY_NAME)"
	@echo ""
	@echo "Make sure $(INSTALL_DIR) is in your PATH."

# Show help
help:
	@echo "tcc-helper Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  make              - Build with automatic signing"
	@echo "  make dev          - Build with debug output"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make install      - Install to ~/bin"
	@echo "  make help         - Show this help"
	@echo ""
	@echo "Code Signing:"
	@echo "  The build automatically uses:"
	@echo "  1. Developer ID Application certificate (if available in keychain)"
	@echo "  2. Stable ad-hoc signature with fixed identifier (fallback)"
	@echo ""
	@echo "  This ensures that permissions granted to tcc-helper persist"
	@echo "  across rebuilds, avoiding the need to re-grant Accessibility"
	@echo "  permission every time you rebuild during development."
	@echo ""
	@echo "Environment Variables:"
	@echo "  MACGO_DEBUG=1              - Enable debug logging"
	@echo "  MACGO_CODE_SIGN_IDENTITY   - Override signing identity"
	@echo "  MACGO_KEEP_BUNDLE=1        - Keep bundle after execution (for inspection)"
