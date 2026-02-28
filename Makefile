BINARY := lotus
BINARY_CLI := lotus-tui
CMD := ./cmd/lotus
CMD_CLI := ./cmd/lotus-tui
BUILD_DIR := ./build
OUT := $(BUILD_DIR)/$(BINARY)
OUT_CLI := $(BUILD_DIR)/$(BINARY_CLI)

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION := $(shell go version | cut -d' ' -f3)
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME) -X main.goVersion=$(GO_VERSION)"

DB_PATH := $(HOME)/.local/share/lotus/lotus.duckdb

.PHONY: build build-lotus build-cli run run-cli dev test clean prune

build: build-lotus build-cli

build-lotus:
	@mkdir -p $(BUILD_DIR)
	@go build -trimpath $(LDFLAGS) -o $(OUT) $(CMD)
	@echo "built $(OUT)"

build-cli:
	@mkdir -p $(BUILD_DIR)
	@go build -trimpath $(LDFLAGS) -o $(OUT_CLI) $(CMD_CLI)
	@echo "built $(OUT_CLI)"

run: build-lotus
	@$(OUT) --config ./cmd/lotus/config.yml

run-cli: build-cli
	@$(OUT_CLI)

dev: build
	@$(OUT) --config ./cmd/lotus/config.yml & LOTUS_PID=$$!; \
	sleep 1; \
	$(OUT_CLI); \
	kill $$LOTUS_PID 2>/dev/null; wait $$LOTUS_PID 2>/dev/null

test:
	@go test ./...

clean:
	@rm -rf $(BUILD_DIR)

prune:
	@rm -f $(DB_PATH) $(DB_PATH).wal
	@echo "pruned $(DB_PATH)"
