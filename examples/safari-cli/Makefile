.PHONY: build install uninstall test clean

BINARY_NAME=safari-cli
INSTALL_PATH=$(HOME)/bin

build:
	go build -o $(BINARY_NAME)

install: build
	mkdir -p $(INSTALL_PATH)
	cp $(BINARY_NAME) $(INSTALL_PATH)/
	@echo "✓ Installed to $(INSTALL_PATH)/$(BINARY_NAME)"
	@echo "Make sure $(INSTALL_PATH) is in your PATH"

uninstall:
	rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✓ Uninstalled $(BINARY_NAME)"

test: build
	./$(BINARY_NAME) list-api
	./$(BINARY_NAME) test

clean:
	rm -f $(BINARY_NAME)

.DEFAULT_GOAL := build
