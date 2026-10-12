.PHONY: format test-tools check-headings check-go check-docs check-bbs-states check
PYTHON ?= python3

format:
	$(PYTHON) tools/format_markdown.py

test-tools:
	$(PYTHON) -m unittest discover -s tests -v

check-headings:
	$(PYTHON) check_heading_structure.py

check-go:
	$(PYTHON) tools/check_go.py $(GO_CHECK_FLAGS)

# Documentation quality gate: bilingual structure, relative links, glossary
# anchors, documented asset paths, and the workshop skeleton from
# docs/TEMPLATE.md. Use together with check-go for a full verification run.
check-docs: check-headings
	$(PYTHON) tools/check_links.py
	$(PYTHON) tools/check_glossary_anchors.py
	$(PYTHON) tools/check_asset_paths.py
	$(PYTHON) tools/check_workshop_skeleton.py
	$(PYTHON) tools/check_go_requirements.py

# Full repository verification (docs + Go modules).
check-bbs-states:
	$(PYTHON) tools/check_bbs_states.py

check: check-docs check-go check-bbs-states
