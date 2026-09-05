"""Discover workshop Markdown without traversing caches or tool metadata."""

import os
from pathlib import Path


EXCLUDED_DIRECTORIES = {
    '.git', '.gemini', '.codex', '.claude', '.serena', '.omc',
    '.gomodcache', '.gocache', '.gopath', '.cache', '.pnpm-store',
    '.venv', 'venv', 'conductor', 'node_modules',
}


def markdown_files(root: Path) -> list[Path]:
    files = []
    for directory, children, names in os.walk(root):
        children[:] = sorted(name for name in children if name not in EXCLUDED_DIRECTORIES)
        files.extend(
            Path(directory) / name
            for name in names
            if name.endswith('.md') and name != 'CLAUDE.md'
            and not (Path(directory) / name).is_symlink()
        )
    return sorted(files)
