
VERSION ?= $(shell cat VERSION 2>/dev/null || echo "0.0.1+dev")
COMMIT_HASH ?= $(shell git describe --always --dirty --tags --long 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date --iso-8601=seconds)
LDFLAGS = -s -X main.version=$(VERSION) -X main.commitHash=$(COMMIT_HASH) -X main.buildTime=$(BUILD_TIME)
GOFLAGS = -ldflags "$(LDFLAGS)"
GO_CMD = CGO_ENABLED=0 go build
BIN_DIR = ./bin
TMP_DIR = ./tmp
APPS_DIR = ./cmd

all: spitfire init

$(BIN_DIR) $(TMP_DIR):
	mkdir -p $@

spitfire:
	$(GO_CMD) $(GOFLAGS) -o $(BIN_DIR)/$@ $(APPS_DIR)/$@

init:
	$(GO_CMD) $(GOFLAGS) -o $(BIN_DIR)/spitfire-$@ $(APPS_DIR)/$@
