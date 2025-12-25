.PHONY: format

format:
	@echo "Formatting markdown files..."
	npx markdownlint "**/*.md" --fix
