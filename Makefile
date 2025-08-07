
VERSION ?= $(shell cat VERSION 2>/dev/null || echo "0.0.1+dev")
COMMIT_HASH ?= $(shell git describe --always --dirty --tags --long 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date --iso-8601=seconds)
LDFLAGS = -s -X main.version=$(VERSION) -X main.commitHash=$(COMMIT_HASH) -X main.buildTime=$(BUILD_TIME)
GOFLAGS = -ldflags "$(LDFLAGS)"
GO_CMD = CGO_ENABLED=0 go build
BIN_DIR = ./bin
TMP_DIR = ./tmp
APPS_DIR = ./cmd
APP ?= spitfire
MAIN = $(addprefix cmd/, $(APP))
BIN = $(addprefix bin/, $(APP))

# all: spitfire init clean test lint
.PHONY: clean test lint install uninstall
.DEFAULT_GOAL := $(BIN)

# Installation paths
PREFIX ?= /usr/local
INSTALL_DIR = $(PREFIX)/bin

$(BIN_DIR) $(TMP_DIR):
	mkdir -p $@

$(BIN): $(wildcard $(MAIN)/*.go)
	$(GO_CMD) $(GOFLAGS) -o $@ ./$(MAIN)

# $(BIN_DIR)/spitfire:
# 	$(GO_CMD) $(GOFLAGS) -o $(BIN_DIR)/$@ $(APPS_DIR)/$@

clean:
	rm -rf $(BIN)

test:
	go test -v ./...

# Basic linting
lint:
	go vet ./...
	go fmt ./...

deps:
	go mod download
	go mod tidy

install: $(BIN)
	@echo "Installing spitfire to $(INSTALL_DIR)"
	sudo mkdir -p $(INSTALL_DIR)
	sudo cp $(BIN) $(INSTALL_DIR)/spitfire
	sudo chmod +x $(INSTALL_DIR)/spitfire
	@echo "Installation complete! spitfire is now available system-wide"
	@echo "Try: spitfire --version"

uninstall:
	@echo "Removing spitfire from $(INSTALL_DIR)"
	sudo rm -f $(INSTALL_DIR)/spitfire
	@echo "Uninstallation complete!"

info:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT_HASH)"
	@echo "Build Time: $(BUILD_TIME)"
