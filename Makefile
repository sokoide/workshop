.PHONY: format test-tools check-headings check-go
PYTHON ?= python3

format:
	$(PYTHON) tools/format_markdown.py

test-tools:
	$(PYTHON) -m unittest discover -s tests -v

check-headings:
	$(PYTHON) check_heading_structure.py

check-go:
	$(PYTHON) tools/check_go.py $(GO_CHECK_FLAGS)
