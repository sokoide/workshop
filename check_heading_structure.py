#!/usr/bin/env python3
"""
Check heading structure synchronization between en/ja bilingual pairs.
Extracts heading hierarchy (# ## ###) and compares section counts.
"""

import re
from pathlib import Path
from typing import List, Tuple, Dict
from collections import defaultdict


def extract_headings(content: str) -> List[Tuple[int, str, str]]:
    """
    Extract headings from markdown content.
    Returns list of (level, line_text, heading_text) tuples.
    """
    heading_pattern = re.compile(r'^(#{1,6})\s+(.+)$', re.MULTILINE)
    headings = []
    for match in heading_pattern.finditer(content):
        level = len(match.group(1))
        line_text = match.group(0)
        heading_text = match.group(2)
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
    ja_files = sorted(repo_root.rglob("*_ja.md"))
    en_files = sorted(repo_root.rglob("*_en.md"))
    pairs = []

    # Create a lookup for en files
    en_lookup = {}
    for en_file in en_files:
        if "node_modules" in str(en_file) or ".git" in str(en_file):
            continue
        # For tcpip_stack_en.md, store key as "tcpip_stack"
        key = en_file.stem.replace("_en", "")
        en_lookup[key] = en_file

    for ja_file in ja_files:
        # Skip node_modules and .git
        if "node_modules" in str(ja_file) or ".git" in str(ja_file):
            continue

        # Try to find matching en file
        ja_stem = ja_file.stem.replace("_ja", "")

        # Method 1: Check en_lookup for _en pattern
        if ja_stem in en_lookup:
            pairs.append((en_lookup[ja_stem], ja_file))
            continue

        # Method 2: Check for plain .md pattern (e.g., clean_arch.md)
        en_file = ja_file.parent / f"{ja_stem}.md"
        if en_file.exists() and en_file.is_file():
            pairs.append((en_file, ja_file))
            continue

        # Check if English version exists
        if en_file.exists() and en_file != ja_file:
            pairs.append((en_file, ja_file))

    return pairs


def analyze_pair(en_file: Path, ja_file: Path) -> Dict:
    """Analyze a single en/ja pair."""
    try:
        en_content = en_file.read_text(encoding="utf-8")
        ja_content = ja_file.read_text(encoding="utf-8")
    except Exception as e:
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
        "mismatches": mismatches
    }


def main():
    """Main function to analyze all bilingual pairs."""
    repo_root = Path("/Users/scott/repo/sokoide/workshop")

    print("# Heading Structure Synchronization Report")
    print(f"Generated: 2026-06-07")
    print(f"Repository: {repo_root}")
    print()

    pairs = find_bilingual_pairs(repo_root)
    print(f"Found {len(pairs)} bilingual pairs\n")

    if not pairs:
        print("No bilingual pairs found.")
        return

    # Analyze all pairs
    results = []
    mismatches_found = []

    for en_file, ja_file in pairs:
        result = analyze_pair(en_file, ja_file)
        results.append(result)

        if "error" in result:
            print(f"⚠️  ERROR: {result['en']} / {result['ja']}")
            print(f"   {result['error']}\n")
        elif result["mismatches"]:
            mismatches_found.append(result)
            rel_en = result["en"].replace(str(repo_root) + "/", "")
            rel_ja = result["ja"].replace(str(repo_root) + "/", "")
            print(f"⚠️  MISMATCH: {rel_en} <-> {rel_ja}")

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


if __name__ == "__main__":
    main()
