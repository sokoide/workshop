PRETTIER_IGNORED := conductor node_modules .gomodcache .omc .serena

.PHONY: format

# use pnpm if available, otherwise npx
RUNNER := $(shell command -v pnpm >/dev/null 2>&1 && echo "pnpm dlx" || echo "npx")
EXEC := $(shell command -v pnpm >/dev/null 2>&1 && echo "pnpm exec" || echo "npx")

IGNORE_PATHS := $(foreach dir,$(PRETTIER_IGNORED),! -path "./$(dir)/*")

format:
	@echo "Formatting markdown tables with prettier..."
	@find . -name "*.md" ! -name "CLAUDE.md" $(IGNORE_PATHS) \
	  -exec $(EXEC) prettier --write --parser markdown {} + 2>&1 | grep -v "^$$" | grep -v "(unchanged)" || true
	@echo "Linting markdown files..."
	$(RUNNER) markdownlint-cli "**/*.md" --ignore "conductor/**" --ignore "CLAUDE.md" --ignore "node_modules/**" --ignore ".gomodcache/**" --ignore ".omc/**" --ignore ".serena/**" --fix
	$(EXEC) textlint --fix "**/*.md"
