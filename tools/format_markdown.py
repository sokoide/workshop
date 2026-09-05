"""Run installed formatters on one shared file list, preserving failures."""

from pathlib import Path
import subprocess
import sys

from markdown_files import markdown_files


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    commands = [
        ('prettier', '--write', '--parser', 'markdown'),
        ('markdownlint', '--fix'),
        ('textlint', '--fix'),
    ]
    for name, *_ in commands:
        if not (root / 'node_modules' / '.bin' / name).is_file():
            print(f'Missing {name}: run pnpm install or npm install first.', file=sys.stderr)
            return 1

    files = [str(path.relative_to(root)) for path in markdown_files(root)]
    for name, *options in commands:
        print(f'Running {name} on {len(files)} Markdown files...', flush=True)
        # Keep argument lists bounded even as the workshop collection grows.
        for offset in range(0, len(files), 100):
            result = subprocess.run(
                [str(root / 'node_modules' / '.bin' / name), *options,
                 '--', *files[offset:offset + 100]],
                cwd=root,
            )
            if result.returncode:
                return result.returncode
    return 0


if __name__ == '__main__':
    sys.exit(main())
