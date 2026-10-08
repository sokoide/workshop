"""Validate relative links and image references in workshop Markdown files."""

import re
import sys
from pathlib import Path

from markdown_files import markdown_files


FENCE_PATTERN = re.compile(r'^ {0,3}(`{3,}|~{3,})(.*)$')
LINK_PATTERN = re.compile(r'!?\[[^\]]*\]\(\s*([^)\s]+)(?:\s+"[^"]*")?\s*\)')
EXTERNAL_PATTERN = re.compile(r'^[a-zA-Z][a-zA-Z0-9.+-]*:')


def lines_outside_fences(content: str) -> list[tuple[int, str]]:
    """Return (line number, text) pairs for lines outside fenced code blocks."""
    lines = []
    active_fence = None
    for number, line in enumerate(content.splitlines(), start=1):
        fence = FENCE_PATTERN.match(line)
        if fence:
            marker = fence.group(1)
            character = marker[0]
            if active_fence is None:
                if character != '`' or '`' not in fence.group(2):
                    active_fence = (character, len(marker))
            elif (character == active_fence[0]
                  and len(marker) >= active_fence[1]
                  and not fence.group(2).strip()):
                active_fence = None
            continue
        if active_fence is None:
            lines.append((number, line))
    return lines


def link_violations(path: Path) -> list[tuple[int, str, str]]:
    """Collect (line, target, reason) tuples for broken links in one file."""
    violations = []
    content = path.read_text(encoding='utf-8')
    for number, line in lines_outside_fences(content):
        for match in LINK_PATTERN.finditer(line):
            target = match.group(1)
            # Windows drive letters ("C:\Users\...") and file:// URLs look
            # scheme-prefixed but are machine-local absolute paths.
            if re.match(r'^[a-zA-Z]:[\\/]', target) or target.startswith('file:'):
                violations.append((number, target, 'absolute path'))
                continue
            if EXTERNAL_PATTERN.match(target):
                continue
            if target.startswith('/') or '/Users/' in target or '/home/' in target:
                violations.append((number, target, 'absolute path'))
                continue
            destination = target.split('#', 1)[0].split('?', 1)[0]
            if not destination:
                continue  # Same-page anchor such as (#section).
            if not (path.parent / destination).exists():
                violations.append((number, target, 'missing target'))
    return violations


def report(files: list[Path], root: Path) -> int:
    """Print violations as file:line entries and return the exit status."""
    violations = [
        (path, number, target, reason)
        for path in files
        for number, target, reason in link_violations(path)
    ]
    for path, number, target, reason in violations:
        try:
            location = path.relative_to(root)
        except ValueError:
            location = path
        print(f'{location}:{number}: {reason}: {target}')
    print(f'Files checked: {len(files)}, link violations: {len(violations)}')
    return 1 if violations else 0


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    return report(markdown_files(root), root)


if __name__ == '__main__':
    sys.exit(main())
