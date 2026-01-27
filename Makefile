GO ?= go
BUN ?= bun
BIN_DIR ?= bin
TMP_DIR ?= .tmp

.PHONY: all build server run-server run-server-dev web-ui-install web-ui-build web-ui-dev fmt vet lint test tidy deps clean

all: build

build: server

server: $(BIN_DIR)/server
$(BIN_DIR)/server:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/server ./server

run-server: web-ui-install web-ui-build
	$(GO) run ./server

run-server-dev: web-ui-install web-ui-dev
	$(GO) run . server -config configs/dev.yaml

web-ui-dev: clean
	@cd web-ui && { mkdir -p ../$(TMP_DIR); nohup $(BUN) run dev -- --port 5002 > ../$(TMP_DIR)/hft-webui-dev.log 2>&1 & echo $$! > ../$(TMP_DIR)/hft-webui-dev.pid; }
	@echo "---> Web UI dev server started on :5002 (pid: $$(cat $(TMP_DIR)/hft-webui-dev.pid))"

web-ui-build:
	cd web-ui && $(BUN) run build
	@echo "--->  Web UI built"

web-ui-install:
	cd web-ui && $(BUN) install
	@echo "--->  Web UI installed"

clean: 
	rm -rf $(TMP_DIR)
	rm -rf $(BIN_DIR)

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint: fmt vet

test:
	$(GO) test ./...

tidy:
	$(GO) mod tidy

deps:
	$(GO) mod download


