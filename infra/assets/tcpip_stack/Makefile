# TCP/IP Stack Workshop Makefile

.PHONY: help build run ping clean test deps

help:
	@echo "TCP/IP Protocol Stack Workshop"
	@echo ""
	@echo "Targets:"
	@echo "  make build    - Build the TCP/IP stack"
	@echo "  make run      - Run the TCP/IP stack (requires sudo)"
	@echo "  make ping     - Run custom ping implementation"
	@echo "  make clean    - Remove built binaries"
	@echo "  make test     - Run tests"
	@echo "  make deps     - Download dependencies"

deps:
	@echo "Downloading dependencies..."
	go mod tidy
	go mod download

build: deps
	@echo "Building TCP/IP stack..."
	go build -o tcpip_stack main.go
	@echo "Build complete: ./tcpip_stack"

run: build
	@echo "Starting TCP/IP stack..."
	@echo "Note: This requires raw socket permissions (sudo)"
	sudo ./tcpip_stack -iface eth0

ping: deps
	@echo "Building custom ping..."
	go build -o bin/ping ./cmd/ping
	@echo "Building complete: ./bin/ping"
	@echo "Note: This requires raw socket permissions (sudo)"
	sudo ./bin/ping -iface eth0 -target 192.168.1.1

clean:
	@echo "Cleaning..."
	rm -f tcpip_stack
	rm -rf bin/
	@echo "Clean complete"

test:
	@echo "Running tests..."
	go test -v ./...

# Show available interfaces
interfaces:
	@echo "Available network interfaces:"
	@go run main.go 2>&1 | grep -A 20 "Available interfaces" || true

# Setup network namespace environment for testing
setup-env:
	@echo "Setting up network namespace 'workshop'..."
	sudo ip netns add workshop || true
	sudo ip link add veth-host type veth peer name veth-ns || true
	sudo ip link set veth-ns netns workshop || true
	sudo ip link set veth-host up
	sudo ip netns exec workshop ip link set veth-ns up
	sudo ip addr add 192.168.100.1/24 dev veth-host || true
	sudo ip netns exec workshop ip addr add 192.168.100.2/24 dev veth-ns || true
	@echo "Environment setup complete."
	@echo "veth-host: 192.168.100.1 (Host side)"
	@echo "veth-ns:   192.168.100.2 (Namespace side)"

# Remove network namespace environment
teardown-env:
	@echo "Tearing down network namespace 'workshop'..."
	sudo ip link delete veth-host || true
	sudo ip netns delete workshop || true
	@echo "Teardown complete."

# Run in the veth environment
run-veth: build
	@echo "Starting TCP/IP stack on veth-host..."
	sudo ./tcpip_stack -iface veth-host

# Ping from the namespace to the host stack
ping-test:
	@echo "Pinging host stack (192.168.100.1) from namespace..."
	sudo ip netns exec workshop ping 192.168.100.1

# Build for multiple platforms
build-all: deps
	@echo "Building for multiple platforms..."
	@mkdir -p bin
	GOARCH=amd64 GOOS=linux go build -o bin/tcpip_stack-linux-amd64 main.go
	GOARCH=arm64 GOOS=linux go build -o bin/tcpip_stack-linux-arm64 main.go
	GOARCH=amd64 GOOS=darwin go build -o bin/tcpip_stack-darwin-amd64 main.go
	GOARCH=arm64 GOOS=darwin go build -o bin/tcpip_stack-darwin-arm64 main.go
	@echo "Build complete: bin/"

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Install tools
tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed"
