"""Verify that documented infra/software asset paths exist in the repository."""

import re
import sys
from pathlib import Path

from markdown_files import markdown_files


# Match only repo-rooted references such as "cp -r software/assets/... ";
# relative "assets/..." links are covered by check_links instead.
ASSET_PATH_PATTERN = re.compile(r'\b(?:infra|software)/assets/[A-Za-z0-9_/.-]+')

# Paths that legitimately do not exist until the reader builds/runs an asset
# (clean-up commands such as `rm -f software/assets/bbs/bbs`). Documented on
# purpose, so the existence check must not flag them.
GENERATED_PATHS = {
    'software/assets/bbs/bbs',      # go build artifact
    'software/assets/bbs/bbs.db',   # sqlite file created on first run
}


def strip_suffixes(reference: str) -> str:
    """Drop line numbers (main.go:42), Go wildcards (...), and sentence periods."""
    reference = re.sub(r'(:\d+|\.{3})+$', '', reference)
    return reference.rstrip('.')


def asset_violations(path: Path, root: Path) -> list[tuple[int, str]]:
    """Collect (line, reference) tuples for asset paths missing from root."""
    violations = []
    content = path.read_text(encoding='utf-8')
    for number, line in enumerate(content.splitlines(), start=1):
        for reference in ASSET_PATH_PATTERN.findall(line):
            candidate = strip_suffixes(reference)
            if candidate and candidate not in GENERATED_PATHS \
                    and not (root / candidate).exists():
                violations.append((number, reference))
    return violations


def report(files: list[Path], root: Path) -> int:
    """Print violations as file:line entries and return the exit status."""
    violations = [
        (path, number, reference)
        for path in files
        for number, reference in asset_violations(path, root)
    ]
    for path, number, reference in violations:
        try:
            location = path.relative_to(root)
        except ValueError:
            location = path
        print(f'{location}:{number}: missing asset path: {reference}')
    print(f'Files checked: {len(files)}, asset path violations: {len(violations)}')
    return 1 if violations else 0


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    return report(markdown_files(root), root)


if __name__ == '__main__':
    sys.exit(main())
