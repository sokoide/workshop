BINARY_NAME=bbs
CMD_DIR=./cmd/bbs
DB_FILE=bbs.db

.PHONY: all
all: build

.PHONY: build
build:
	go build -o $(BINARY_NAME) $(CMD_DIR)

.PHONY: run
run: build
	./$(BINARY_NAME)

.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f $(DB_FILE)

.PHONY: format
format:
	@if command -v markdownlint > /dev/null; then \
		markdownlint --fix "**/*.md"; \
	elif command -v pnpm > /dev/null; then \
		pnpm dlx markdownlint-cli --fix "**/*.md"; \
	elif command -v npx > /dev/null; then \
		npx markdownlint-cli --fix "**/*.md"; \
	else \
		echo "markdownlint not found. Please install it or use pnpm/npx."; \
		exit 1; \
	fi

# Pinned tools keep generated source reproducible without changing go.mod.
.PHONY: proto
proto:
	mkdir -p bin/proto
	GOBIN=$(CURDIR)/bin/proto go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.9
	GOBIN=$(CURDIR)/bin/proto go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
	protoc --plugin=protoc-gen-go=$(CURDIR)/bin/proto/protoc-gen-go --plugin=protoc-gen-go-grpc=$(CURDIR)/bin/proto/protoc-gen-go-grpc --go_out=. --go_opt=module=github.com/sokoide/workshop/software/assets/bbs --go-grpc_out=. --go-grpc_opt=module=github.com/sokoide/workshop/software/assets/bbs api/proto/bbs.proto
