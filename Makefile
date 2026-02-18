GO ?= go
BUN ?= bun
BIN_DIR ?= bin
TMP_DIR ?= .tmp

.PHONY: all build server run-server run-server-dev web-ui-install web-ui-build web-ui-dev web-ui-stop fmt vet lint test tidy deps clean

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

web-ui-dev:
	@if [ -f $(TMP_DIR)/hft-webui-dev.pid ]; then \
		kill -9 $$(cat $(TMP_DIR)/hft-webui-dev.pid) 2>/dev/null || true; \
		rm -f $(TMP_DIR)/hft-webui-dev.pid; \
	fi
	@lsof -ti:5002 | xargs kill -9 2>/dev/null || true
	@echo "---> Starting Web UI dev server on port 5002..."
	@cd web-ui && { mkdir -p ../$(TMP_DIR); nohup $(BUN) run dev -- --host 0.0.0.0 --port 5002 > ../$(TMP_DIR)/hft-webui-dev.log 2>&1 & echo $$! > ../$(TMP_DIR)/hft-webui-dev.pid; }
	@echo "---> Waiting for server to start..."
	@sleep 3
	@if lsof -ti:5002 > /dev/null 2>&1; then \
		echo "---> Web UI dev server started successfully on 0.0.0.0:5002 (pid: $$(cat $(TMP_DIR)/hft-webui-dev.pid))"; \
	else \
		echo "---> Warning: Server may not have started on port 5002. Check logs at $(TMP_DIR)/hft-webui-dev.log"; \
	fi

web-ui-build:
	cd web-ui && $(BUN) run build
	@echo "--->  Web UI built"

web-ui-install:
	cd web-ui && $(BUN) install
	@echo "--->  Web UI installed"

web-ui-stop:
	@if [ -f $(TMP_DIR)/hft-webui-dev.pid ]; then \
		kill -9 $$(cat $(TMP_DIR)/hft-webui-dev.pid) 2>/dev/null && echo "---> Web UI dev server stopped (pid: $$(cat $(TMP_DIR)/hft-webui-dev.pid))" || echo "---> No running Web UI dev server found"; \
		rm -f $(TMP_DIR)/hft-webui-dev.pid; \
	else \
		echo "---> No PID file found. Attempting to kill process on port 5002..."; \
		lsof -ti:5002 | xargs kill -9 2>/dev/null && echo "---> Process on port 5002 stopped" || echo "---> No process found on port 5002"; \
	fi

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


