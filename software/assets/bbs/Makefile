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
