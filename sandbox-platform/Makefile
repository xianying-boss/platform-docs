BINARY_API    := bin/api-server
BINARY_AGENT  := bin/node-agent
MODULE        := github.com/sandbox/platform

.PHONY: all build api-server node-agent tidy lint test wasm-tools docker-desktop clean

## Default: build both binaries
all: build

## Build both binaries
build: api-server node-agent

## Build the API server
api-server:
	@mkdir -p bin
	go build -o $(BINARY_API) ./cmd/api-server

## Build the node agent
node-agent:
	@mkdir -p bin
	go build -o $(BINARY_AGENT) ./cmd/node-agent

## Fetch dependencies
tidy:
	go mod tidy

## Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

## Run unit tests
test:
	go test -race -count=1 ./...

## Build WASM tools (requires wasip1 target)
wasm-tools:
	@echo "Building WASM tools..."
	@for tool in html_parse json_parse markdown_convert docx_generate; do \
		echo "  Building $$tool.wasm"; \
		GOOS=wasip1 GOARCH=wasm go build \
			-o /var/sandbox/wasm-modules/$$tool.wasm \
			./../../sandbox-tools/wasm/$$tool/ || true; \
	done

## Build Docker desktop-runner image
docker-desktop:
	docker build -t sandbox/desktop-runner:latest ./docker/desktop-runner

## Build Firecracker snapshot (requires kernel + rootfs)
## Usage: make snapshot KERNEL=/path/vmlinux ROOTFS=/path/rootfs.ext4
snapshot:
	@[[ -n "$(KERNEL)" ]] || (echo "KERNEL= required" && exit 1)
	@[[ -n "$(ROOTFS)" ]] || (echo "ROOTFS= required" && exit 1)
	bash scripts/build-snapshot.sh --kernel $(KERNEL) --rootfs $(ROOTFS)

## Start local dev services
dev-up:
	docker compose -f infra/docker-compose.yml up -d

## Stop local dev services
dev-down:
	docker compose -f infra/docker-compose.yml down

## Run API server in dev mode (no JWT, no config file)
run-api:
	./$(BINARY_API)

## Run node agent (dev mode, node-id=dev-node-1)
run-agent:
	./$(BINARY_AGENT) --node-id dev-node-1 --data-dir /tmp/sandbox-data

## Quick smoke test
ping:
	curl -s http://localhost:8080/health | python3 -m json.tool

test-job:
	curl -s -X POST http://localhost:8080/v1/execute \
		-H "Content-Type: application/json" \
		-d '{"tool":"python_run","input":{"code":"print(1+1)"}}' | python3 -m json.tool

clean:
	rm -rf bin/
