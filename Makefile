.PHONY: format

# use pnpm if available, otherwise npx
RUNNER := $(shell command -v pnpm >/dev/null 2>&1 && echo "pnpm dlx" || echo "npx")

format:
	@echo "Formatting markdown files using $(RUNNER)..."
	$(RUNNER) markdownlint-cli "**/*.md" --ignore "conductor/**" --fix
	$(RUNNER) textlint --fix "**/*.md"

