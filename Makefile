.PHONY: format

format:
	@echo "Formatting markdown files..."
	npx markdownlint "**/*.md" --ignore "conductor/**" --fix
	npx textlint --fix "**/*.md"

