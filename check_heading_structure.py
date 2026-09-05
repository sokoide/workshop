#!/usr/bin/env python3
"""
Check heading structure synchronization between en/ja bilingual pairs.
Compares ATX heading level order (# ## ###), with section counts for diagnostics.
"""

import re
import sys
from datetime import datetime
from pathlib import Path
from typing import List, Tuple, Dict
from collections import defaultdict
from tools.markdown_files import markdown_files


def extract_headings(content: str) -> List[Tuple[int, str, str]]:
    """
    Extract headings from markdown content.
    Returns list of (level, line_text, heading_text) tuples.
    """
    heading_pattern = re.compile(r'^ {0,3}(#{1,6})(?:[ \t]+(.*)|$)')
    fence_pattern = re.compile(r'^ {0,3}(`{3,}|~{3,})(.*)$')
    headings = []
    active_fence = None

    for line in content.splitlines():
        fence_match = fence_pattern.match(line)
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

        if active_fence is not None:
            continue

        match = heading_pattern.match(line)
        if not match:
            continue
        level = len(match.group(1))
        line_text = line
        heading_text = match.group(2) or ''
        headings.append((level, line_text, heading_text))
    return headings


def count_sections(headings: List[Tuple[int, str, str]]) -> Dict[int, int]:
    """Count headings by level."""
    counts = defaultdict(int)
    for level, _, _ in headings:
        counts[level] += 1
    return dict(counts)


def find_bilingual_pairs(repo_root: Path) -> List[Tuple[Path, Path]]:
    """Find en/ja markdown file pairs."""
    ja_files = [path for path in markdown_files(repo_root) if path.name.endswith('_ja.md')]
    pairs = []

    for ja_file in ja_files:
        ja_stem = ja_file.stem.removesuffix("_ja")

        # Prefer the sibling *_en.md file. Keeping the lookup directory-local
        # avoids pairing files with identical stems from different workshops.
        en_file = ja_file.with_name(f"{ja_stem}_en.md")
        if en_file.is_file():
            pairs.append((en_file, ja_file))
            continue

        # Some pairs use an unsuffixed English file (for example README.md).
        en_file = ja_file.parent / f"{ja_stem}.md"
        if en_file.is_file():
            pairs.append((en_file, ja_file))

    return pairs


def analyze_pair(en_file: Path, ja_file: Path) -> Dict:
    """Analyze a single en/ja pair."""
    try:
        en_content = en_file.read_text(encoding="utf-8")
        ja_content = ja_file.read_text(encoding="utf-8")
    except (OSError, UnicodeError) as e:
        return {
            "en": str(en_file),
            "ja": str(ja_file),
            "error": str(e)
        }

    en_headings = extract_headings(en_content)
    ja_headings = extract_headings(ja_content)

    en_sections = count_sections(en_headings)
    ja_sections = count_sections(ja_headings)

    # Check for mismatches
    mismatches = []
    for level in range(1, 7):
        en_count = en_sections.get(level, 0)
        ja_count = ja_sections.get(level, 0)
        if en_count != ja_count:
            mismatches.append({
                "level": level,
                "en_count": en_count,
                "ja_count": ja_count,
                "diff": ja_count - en_count
            })

    return {
        "en": str(en_file),
        "ja": str(ja_file),
        "en_headings": len(en_headings),
        "ja_headings": len(ja_headings),
        "en_sections": en_sections,
        "ja_sections": ja_sections,
        "structure_mismatch": [h[0] for h in en_headings] != [h[0] for h in ja_headings],
        "mismatches": mismatches
    }


def main():
    """Main function to analyze all bilingual pairs."""
    repo_root = Path(__file__).resolve().parent

    print("# Heading Structure Synchronization Report")
    print(f"Generated: {datetime.now().astimezone().isoformat(timespec='seconds')}")
    print(f"Repository: {repo_root}")
    print()

    pairs = find_bilingual_pairs(repo_root)
    print(f"Found {len(pairs)} bilingual pairs\n")

    if not pairs:
        print("No bilingual pairs found.")
        return 0

    # Analyze all pairs
    results = []
    mismatches_found = []

    for en_file, ja_file in pairs:
        result = analyze_pair(en_file, ja_file)
        results.append(result)

        if "error" in result:
            print(f"⚠️  ERROR: {result['en']} / {result['ja']}")
            print(f"   {result['error']}\n")
        elif result["structure_mismatch"]:
            mismatches_found.append(result)
            rel_en = result["en"].replace(str(repo_root) + "/", "")
            rel_ja = result["ja"].replace(str(repo_root) + "/", "")
            print(f"⚠️  MISMATCH: {rel_en} <-> {rel_ja}")
            if not result['mismatches']:
                print('   Heading counts match, but heading level order differs.')

            for mm in result["mismatches"]:
                level_marks = "#" * mm["level"]
                print(f"   Level {mm['level']} ({level_marks}): EN={mm['en_count']}, JA={mm['ja_count']} (diff: {mm['diff']:+d})")
            print()

    # Summary
    print("=" * 80)
    print(f"# Summary")
    print(f"Total pairs analyzed: {len(results)}")
    print(f"Pairs with mismatches: {len(mismatches_found)}")
    print(f"Pairs synchronized: {len(results) - len(mismatches_found) - len([r for r in results if 'error' in r])}")

    # Detailed report for mismatched pairs
    if mismatches_found:
        print("\n" + "=" * 80)
        print("# Detailed Mismatch Report")

        for result in mismatches_found:
            rel_en = result["en"].replace(str(repo_root) + "/", "")
            rel_ja = result["ja"].replace(str(repo_root) + "/", "")

            print(f"\n## {rel_en} <-> {rel_ja}")
            print(f"EN headings: {result['en_headings']}, JA headings: {result['ja_headings']}")

            print("\n### Section Count Comparison")
            print("| Level | EN Count | JA Count | Diff |")
            print("|-------|----------|----------|------|")

            for level in range(1, 7):
                en_count = result["en_sections"].get(level, 0)
                ja_count = result["ja_sections"].get(level, 0)
                diff = ja_count - en_count
                indicator = " 🔴" if diff != 0 else ""
                print(f"| {level} | {en_count} | {ja_count} | {diff:+d} |{indicator}")

    return 1 if mismatches_found or any('error' in result for result in results) else 0


if __name__ == "__main__":
    sys.exit(main())
