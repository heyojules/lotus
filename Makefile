BINARY := tiny-telemetry
BINARY_CLI := tiny-telemetry-tui
CMD := ./cmd/tiny-telemetry
CMD_CLI := ./cmd/tiny-telemetry-tui
BUILD_DIR := ./build
OUT := $(BUILD_DIR)/$(BINARY)
OUT_CLI := $(BUILD_DIR)/$(BINARY_CLI)

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION := $(shell go version | cut -d' ' -f3)
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME) -X main.goVersion=$(GO_VERSION)"

DB_PATH := $(HOME)/.local/share/tiny-telemetry/tiny-telemetry.duckdb

.PHONY: build build-server build-cli run run-cli dev test clean prune

build: build-server build-cli

build-server:
	@mkdir -p $(BUILD_DIR)
	@go build -trimpath $(LDFLAGS) -o $(OUT) $(CMD)
	@echo "built $(OUT)"

build-cli:
	@mkdir -p $(BUILD_DIR)
	@go build -trimpath $(LDFLAGS) -o $(OUT_CLI) $(CMD_CLI)
	@echo "built $(OUT_CLI)"

run: build-server
	@$(OUT) --config ./cmd/tiny-telemetry/config.yml

run-cli: build-cli
	@$(OUT_CLI)

dev: build
	@$(OUT) --config ./cmd/tiny-telemetry/config.yml & TT_PID=$$!; \
	sleep 1; \
	$(OUT_CLI); \
	kill $$TT_PID 2>/dev/null; wait $$TT_PID 2>/dev/null

test:
	@go test ./...

clean:
	@rm -rf $(BUILD_DIR)

prune:
	@rm -f $(DB_PATH) $(DB_PATH).wal
	@echo "pruned $(DB_PATH)"
