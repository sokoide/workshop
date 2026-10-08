"""Validate anchor links pointing at glossary_en.md / glossary_ja.md entries."""

import os
import re
import sys
from pathlib import Path

from check_links import lines_outside_fences
from markdown_files import markdown_files


HEADING_PATTERN = re.compile(r'^ {0,3}(#{1,6})\s+(.+)$')
ANCHOR_TAG_PATTERN = re.compile(r'<a name="([^"]+)"')
REFERENCE_PATTERN = re.compile(r'([^\s()\[\]<>`\'"]*glossary_(?:en|ja)\.md)#([\w-]+)')


def github_slug(text: str) -> str:
    """Approximate GitHub heading slugs for both English and Japanese text."""
    text = re.sub(r'<[^>]+>', '', text).strip().lower()
    text = re.sub(r'[^\w\s-]', '', text)
    return re.sub(r'\s', '-', text)


def glossary_anchors(path: Path) -> set[str]:
    """Collect valid anchors: <a name> values plus heading-derived slugs."""
    anchors = set()
    for _, line in lines_outside_fences(path.read_text(encoding='utf-8')):
        anchors.update(ANCHOR_TAG_PATTERN.findall(line))
        heading = HEADING_PATTERN.match(line)
        if heading:
            slug = github_slug(heading.group(2))
            if slug:
                anchors.add(slug)
    return anchors


def anchor_violations(path: Path,
                      cache: dict[Path, set[str] | None]) -> list[tuple[int, str, str]]:
    """Collect (line, reference, reason) tuples for broken glossary anchors."""
    violations = []
    content = path.read_text(encoding='utf-8')
    for number, line in lines_outside_fences(content):
        for match in REFERENCE_PATTERN.finditer(line):
            reference, anchor = match.groups()
            glossary = Path(os.path.normpath(path.parent / reference))
            if glossary not in cache:
                cache[glossary] = glossary_anchors(glossary) if glossary.is_file() else None
            if cache[glossary] is None:
                violations.append((number, match.group(0), 'glossary file not found'))
            elif anchor not in cache[glossary]:
                violations.append((number, match.group(0), 'unknown anchor'))
    return violations


def report(files: list[Path], root: Path) -> int:
    """Print violations as file:line entries and return the exit status."""
    cache: dict[Path, set[str] | None] = {}
    violations = [
        (path, number, reference, reason)
        for path in files
        for number, reference, reason in anchor_violations(path, cache)
    ]
    for path, number, reference, reason in violations:
        try:
            location = path.relative_to(root)
        except ValueError:
            location = path
        print(f'{location}:{number}: {reason}: {reference}')
    print(f'Files checked: {len(files)}, glossary anchor violations: {len(violations)}')
    return 1 if violations else 0


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    return report(markdown_files(root), root)


if __name__ == '__main__':
    sys.exit(main())
