#!/usr/bin/env python3
"""
Fix code fence language tag inconsistencies in markdown files.
Detects and fixes mismatched opening/closing language tags (e.g., ```bash...```text).
"""

import re
from pathlib import Path
from typing import List, Tuple


def find_markdown_files(directories: List[str]) -> List[Path]:
    """Find all .md files in specified directories."""
    md_files = []
    for directory in directories:
        path = Path(directory)
        if path.exists():
            md_files.extend(path.rglob("*.md"))
    return md_files


def check_fence_mismatches(content: str) -> List[Tuple[int, str, str]]:
    """
    Check for code fence mismatches.
    Returns list of tuples: (line_num, opening_tag, actual_closing)
    """
    fence_pattern = re.compile(r"^```(\w*)\s*$")
    lines = content.split("\n")
    mismatches = []
    stack = []  # (line_num, opening_tag)

    for i, line in enumerate(lines, 1):
        match = fence_pattern.match(line)
        if match:
            lang = match.group(1)
            if stack:
                # Closing fence
                _, opening_lang = stack.pop()
                if lang and lang != opening_lang:
                    mismatches.append((i, opening_lang, lang))
            else:
                # Opening fence
                stack.append((i, lang))

    return mismatches


def fix_fence_mismatches(content: str) -> Tuple[str, List[Tuple[int, str, str]]]:
    """
    Fix code fence mismatches by correcting closing tags.
    Returns (fixed_content, list_of_fixes_applied)
    """
    fence_pattern = re.compile(r"^```(\w*)\s*$")
    lines = content.split("\n")
    fixes = []
    stack = []  # (line_num, opening_tag)

    for i, line in enumerate(lines):
        match = fence_pattern.match(line)
        if match:
            lang = match.group(1)
            if stack:
                # Closing fence
                _, opening_lang = stack.pop()
                if lang and lang != opening_lang:
                    # Fix the closing tag
                    lines[i] = f"```{opening_lang}"
                    fixes.append((i + 1, opening_lang, lang))
            else:
                # Opening fence
                stack.append((i, lang))

    return "\n".join(lines), fixes


def main():
    """Main function to check and fix markdown files."""
    directories = ["infra", "software"]
    md_files = find_markdown_files(directories)

    if not md_files:
        print("No markdown files found in specified directories.")
        return

    print(f"Found {len(md_files)} markdown files to check.\n")

    total_mismatches = 0
    files_with_issues = []

    # Check phase
    for md_file in md_files:
        try:
            content = md_file.read_text(encoding="utf-8")
            mismatches = check_fence_mismatches(content)

            if mismatches:
                files_with_issues.append((md_file, mismatches))
                total_mismatches += len(mismatches)
        except Exception as e:
            print(f"Error reading {md_file}: {e}")

    print(f"Found {total_mismatches} fence tag mismatches in {len(files_with_issues)} files.\n")

    if not files_with_issues:
        print("No issues found. All files are clean!")
        return

    # Fix phase
    for md_file, mismatches in files_with_issues:
        print(f"\n{md_file}:")
        for line_num, opening, actual in mismatches:
            print(f"  Line {line_num}: Opening ```{opening} but closing ```{actual}")

        try:
            content = md_file.read_text(encoding="utf-8")
            fixed_content, fixes = fix_fence_mismatches(content)

            if fixes:
                md_file.write_text(fixed_content, encoding="utf-8")
                print(f"  Fixed {len(fixes)} mismatches.")
        except Exception as e:
            print(f"  Error fixing {md_file}: {e}")

    print(f"\n✓ Fixed {total_mismatches} fence tag mismatches across {len(files_with_issues)} files.")


if __name__ == "__main__":
    main()
