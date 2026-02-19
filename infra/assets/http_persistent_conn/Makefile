.PHONY: run setup gen

# SETUP: Install necessary tools (requires Go)
setup:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
	go install github.com/containers/podman-compose@latest

# CERT: Generate self-signed certificate for HTTP/2
cert:
	openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes -subj "/CN=localhost"

# GENERATE: Generate gRPC code from proto
gen:
	mkdir -p pb
	protoc --go_out=. --go_opt=paths=source_relative 
	    --go-grpc_out=. --go-grpc_opt=paths=source_relative 
	    proto/greeter.proto

# RUN: Start the server
run:
	go mod tidy
	go run main.go

# TEST: Test with curl
test-http1:
	curl -v http://localhost:8080/ http://localhost:8080/

# TEST: Test with websocat
test-ws:
	echo "Hello WebSocket" | websocat -n1 ws://localhost:8080/ws

# TEST: Test with grpcurl
test-grpc:
	grpcurl -plaintext -d '{"name": "workshop"}' localhost:50051 pb.Greeter/SayHello
	grpcurl -plaintext -d '{"name": "workshop"}' localhost:50051 pb.Greeter/SayHelloStream
