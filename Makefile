.PHONY: format

# use pnpm if available, otherwise npx
RUNNER := $(shell command -v pnpm >/dev/null 2>&1 && echo "pnpm dlx" || echo "npx")
EXEC := $(shell command -v pnpm >/dev/null 2>&1 && echo "pnpm exec" || echo "npx")

format:
	@echo "Formatting markdown files using $(RUNNER)..."
	$(RUNNER) markdownlint-cli "**/*.md" --ignore "conductor/**" --ignore "CLAUDE.md" --ignore "node_modules/**" --ignore ".gomodcache/**" --fix
	$(EXEC) textlint --fix "**/*.md"
	@echo "Aligning markdown tables..."
	@find . -name "*.md" \
	  ! -path "./conductor/*" \
	  ! -path "./node_modules/*" \
	  ! -path "./.gomodcache/*" \
	  ! -name "CLAUDE.md" \
	  -exec sh -c 'for f; do nvim --headless -c "MdTableAlignAll" -c "wq" "$$f"; done' _ {} +
