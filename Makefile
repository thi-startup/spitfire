
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
.PHONY: clean test lint
.DEFAULT_GOAL := $(BIN)

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

info:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT_HASH)"
	@echo "Build Time: $(BUILD_TIME)"
