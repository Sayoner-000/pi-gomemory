.PHONY: all build install doctor dist clean

BINARY_NAME=pi-gomemory
BIN_DIR=bin
DIST_DIR=dist
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "v1.0.0")

all: build

build:
	@mkdir -p $(BIN_DIR)
	go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME)-setup .

install: build
	@mkdir -p $(HOME)/.local/bin
	cp $(BIN_DIR)/$(BINARY_NAME)-setup $(HOME)/.local/bin/pi-gomemory-setup
	./$(BIN_DIR)/$(BINARY_NAME)-setup install

doctor: build
	./$(BIN_DIR)/$(BINARY_NAME)-setup doctor

dist:
	@bash scripts/build-release.sh $(VERSION)

clean:
	rm -rf $(BIN_DIR)/$(BINARY_NAME)-setup $(DIST_DIR)
