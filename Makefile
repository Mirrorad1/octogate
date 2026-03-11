.PHONY: build-cli build-hub clean test install-cli dev-cli dev-hub help

help:
	@echo "octogate — x402 CLI + Hub"
	@echo ""
	@echo "Commands:"
	@echo "  make build-cli       Build x402 CLI binary"
	@echo "  make build-hub       Build octogate Hub server"
	@echo "  make clean           Remove build artifacts"
	@echo "  make test            Run all tests"
	@echo "  make install-cli     Install CLI to ~/bin/x402"
	@echo "  make dev-cli         Run CLI in dev mode"
	@echo "  make dev-hub         Run Hub in dev mode"

build-cli:
	@echo "Building x402 CLI..."
	go build -o ./bin/x402 ./cmd/x402

build-hub:
	@echo "Building octogate Hub..."
	go build -o ./bin/octogate-hub ./cmd/hub

clean:
	@echo "Cleaning..."
	rm -rf ./bin

test:
	@echo "Running tests..."
	go test -timeout 60s ./...

install-cli: build-cli
	@echo "Installing x402 to ~/bin/..."
	mkdir -p ~/bin
	cp ./bin/x402 ~/bin/x402
	chmod +x ~/bin/x402
	@echo "✓ x402 installed to ~/bin/x402"

dev-cli:
	@echo "Running CLI dev mode..."
	go run ./cmd/x402 $(ARGS)

dev-hub:
	@echo "Running Hub dev mode (port 8080)..."
	go run ./cmd/hub -port 8080
