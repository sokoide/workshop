#!/usr/bin/env python3
"""
Check workshop document skeletons (*_en.md / *_ja.md pairs) against docs/TEMPLATE.md.

Per file, this reports missing skeleton items:
  1. header blockquote containing ⏱️ (estimated time) and 🎯 (level)
  2. required H2 sections: Goal / Prerequisites / What you learned / Cleanup / References
  3. at least one ✅ checkpoint line outside code fences
  4. at least one expected-output comment (`# Expected output:`, `# 期待される結果:` ...)

Heading wording varies across workshops, so each section accepts the synonyms
observed in the corpus (for example Goal|Goals|Purpose, ゴール|目的). The
patterns favor recall over precision: a wording drift that escapes detection is
preferable to a false violation.

Reference-style documents in EXEMPT_FILES are out of scope. This tool only
reports violations; fixing them is triaged separately.
"""

import re
import sys
from datetime import datetime
from pathlib import Path

# Allow both `python3 tools/check_workshop_skeleton.py` and import from tests.
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from tools.markdown_files import markdown_files


# Reference documents (and the repo README) that do not follow the workshop
# skeleton. Membership is triaged by the maintainers; this list is only a
# starting point.
EXEMPT_FILES = {
    'README.md', 'README_ja.md',
    'TEMPLATE.md',
    'glossary_en.md', 'glossary_ja.md',
    'clean_arch_en.md', 'clean_arch_ja.md',
    'go_cache_patterns_en.md', 'go_cache_patterns_ja.md',
    'go_mq_patterns_en.md', 'go_mq_patterns_ja.md',
    'sharding_en.md', 'sharding_ja.md',
}

# Synonyms per language, normalized to the TEMPLATE.md section names.
# Matched with re.search against the heading text after stripping leading
# decorations (emoji, punctuation), so `## 🧹 Cleanup` counts as Cleanup.
SECTION_PATTERNS = {
    'Goal': {
        'en': re.compile(r'goals?\b|purpose|objective', re.IGNORECASE),
        'ja': re.compile(r'ゴール|目的'),
    },
    'Prerequisites': {
        'en': re.compile(r'prerequisites|preparation|setup', re.IGNORECASE),
        'ja': re.compile(r'前提知識|前提条件|準備'),
    },
    'What you learned': {
        'en': re.compile(r'what you learned|summary|key points', re.IGNORECASE),
        'ja': re.compile(r'学んだこと|まとめ|ポイント'),
    },
    'Cleanup': {
        'en': re.compile(r'cleanup', re.IGNORECASE),
        'ja': re.compile(r'片付け|クリーンアップ'),
    },
    'References': {
        'en': re.compile(r'references', re.IGNORECASE),
        'ja': re.compile(r'参考文献|参考リンク|参考'),
    },
}

FENCE_PATTERN = re.compile(r'^ {0,3}(`{3,}|~{3,})(.*)$')
H2_PATTERN = re.compile(r'^ {0,3}##[ \t]+(.+)$')
DECORATION_PATTERN = re.compile(r'^[\W_]+', re.UNICODE)

# Expected-output comments live inside code fences and mix English and
# Japanese wordings; keep this permissive to avoid false violations.
# Accepted conventions: "# Expected output:", "# Result:", "# Output:",
# "# Example output:", "# 期待される結果:", "# 出力:", "# 出力例:" and close
# variants. "Example output" is accepted because environment-dependent
# samples (e.g. scanner reports) are legitimately labeled as examples.
EXPECTED_OUTPUT_PATTERN = re.compile(
    r'^\s*#.*(?:expected\b|example\b|期待|出力例?\s*[:：]|result\s*[:：]|output\s*[:：]|結果\s*[:：])',
    re.IGNORECASE)


def lines_outside_fences(content: str) -> list[str]:
    """Return content lines with fenced code blocks removed."""
    prose = []
    active_fence = None

    for line in content.splitlines():
        fence_match = FENCE_PATTERN.match(line)
        if fence_match:
            marker = fence_match.group(1)
            marker_char = marker[0]
            if active_fence is None:
                if marker_char != '`' or '`' not in fence_match.group(2):
                    active_fence = (marker_char, len(marker))
            elif (marker_char == active_fence[0]
                  and len(marker) >= active_fence[1]
                  and not fence_match.group(2).strip()):
                active_fence = None
            continue

        if active_fence is None:
            prose.append(line)

    return prose


def normalize_heading(text: str) -> str:
    """Strip leading decorations (emoji, punctuation) from a heading."""
    return DECORATION_PATTERN.sub('', text.strip())


def find_bilingual_pairs(repo_root: Path) -> list[tuple[Path, Path]]:
    """Find (en, ja) markdown pairs, including unsuffixed en files like README.md."""
    pairs = []
    for ja_file in (path for path in markdown_files(repo_root)
                    if path.name.endswith('_ja.md')):
        stem = ja_file.stem.removesuffix('_ja')
        en_file = ja_file.with_name(f'{stem}_en.md')
        if not en_file.is_file():
            en_file = ja_file.parent / f'{stem}.md'
        if en_file.is_file():
            pairs.append((en_file, ja_file))
    return sorted(pairs)


def check_skeleton(content: str, lang: str) -> list[str]:
    """Return the missing skeleton items for a single document."""
    missing = []
    prose = lines_outside_fences(content)

    # 1. Metadata header: the metadata blockquote sits before the first H2.
    header = []
    for line in prose:
        if H2_PATTERN.match(line):
            break
        header.append(line)
    quote_lines = [line for line in header if line.lstrip().startswith('>')]
    if not any('⏱' in line for line in quote_lines):
        missing.append('header ⏱️ (estimated time)')
    if not any('🎯' in line for line in quote_lines):
        missing.append('header 🎯 (level)')

    # 2. Required H2 sections (wording synonyms absorbed per language).
    h2_texts = [normalize_heading(m.group(1))
                for line in prose
                if (m := H2_PATTERN.match(line))]
    for section, patterns in SECTION_PATTERNS.items():
        if not any(patterns[lang].search(text) for text in h2_texts):
            missing.append(f'section: {section}')

    # 3. Checkpoint: at least one ✅ line outside code fences.
    if not any('✅' in line for line in prose):
        missing.append('checkpoint (✅ line)')

    # 4. Expected output: at least one comment line, fences included.
    # Lecture-style documents without a single runnable command (no bash
    # fence) have nothing to annotate, so the requirement does not apply.
    has_bash_fence = any(
        FENCE_PATTERN.match(line)
        and line.lstrip(' \t').startswith(('```bash', '```shell',
                                           '```sh', '```console'))
        for line in content.splitlines())
    if has_bash_fence and not any(
            EXPECTED_OUTPUT_PATTERN.match(line)
            for line in content.splitlines()):
        missing.append('expected output comment')

    return missing


def main() -> int:
    """Run the skeleton check over all bilingual pairs and report violations."""
    repo_root = Path(__file__).resolve().parents[1]

    print('# Workshop Skeleton Report')
    print(f'Generated: {datetime.now().astimezone().isoformat(timespec="seconds")}')
    print(f'Repository: {repo_root}')
    print()

    pairs = find_bilingual_pairs(repo_root)
    exempt = sum(1 for en, ja in pairs for f in (en, ja) if f.name in EXEMPT_FILES)
    print(f'Found {len(pairs)} bilingual pairs '
          f'({len(pairs) * 2 - exempt} checked, {exempt} exempt)')
    print()

    checked = 0
    violation_count = 0
    missing_total = 0

    for en_file, ja_file in pairs:
        for path, lang in ((en_file, 'en'), (ja_file, 'ja')):
            if path.name in EXEMPT_FILES:
                continue
            checked += 1
            try:
                missing = check_skeleton(
                    path.read_text(encoding='utf-8'), lang)
            except (OSError, UnicodeError) as e:
                missing = [f'read error: {e}']

            if missing:
                violation_count += 1
                missing_total += len(missing)
                rel = str(path).replace(str(repo_root) + '/', '')
                print(f'⚠️  VIOLATION: {rel}')
                for item in missing:
                    print(f'   - {item}')
                print()

    print('=' * 80)
    print('# Summary')
    print(f'Files checked: {checked}')
    print(f'Files with violations: {violation_count}')
    print(f'Total missing items: {missing_total}')

    return 1 if violation_count else 0


if __name__ == '__main__':
    sys.exit(main())
