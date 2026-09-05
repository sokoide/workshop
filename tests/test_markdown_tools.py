import contextlib
import io
from pathlib import Path
import tempfile
import unittest
import shutil
import subprocess
from unittest.mock import patch

import check_heading_structure as headings


class HeadingTests(unittest.TestCase):
    def test_tagged_fence_does_not_close_code_block(self):
        content = '# Title\n```bash\n```text\n# Not a heading\n```\n## Section\n'
        self.assertEqual([h[0] for h in headings.extract_headings(content)], [1, 2])

    def test_long_fences_and_indented_headings(self):
        content = ' # Title\n````markdown\n```\n# Example\n````\n  ## Section\n'
        self.assertEqual([h[0] for h in headings.extract_headings(content)], [1, 2])

    def test_heading_order_is_checked(self):
        with tempfile.TemporaryDirectory() as directory:
            en = Path(directory) / 'topic_en.md'
            ja = Path(directory) / 'topic_ja.md'
            en.write_text('# Title\n## One\n### Child\n## Two\n')
            ja.write_text('# Title\n## One\n## Two\n### Child\n')
            result = headings.analyze_pair(en, ja)
            self.assertTrue(result['structure_mismatch'])
            self.assertEqual(result['mismatches'], [])

    def test_discovery_prunes_exact_directory_names(self):
        with tempfile.TemporaryDirectory(prefix='project.git-') as directory:
            root = Path(directory)
            for name in ('workshop', 'node_modules', '.gomodcache', 'conductor'):
                folder = root / name
                folder.mkdir()
                (folder / 'topic_en.md').touch()
                (folder / 'topic_ja.md').touch()
            self.assertEqual(headings.find_bilingual_pairs(root), [
                (root / 'workshop/topic_en.md', root / 'workshop/topic_ja.md')
            ])

    def test_cli_exit_status(self):
        with tempfile.TemporaryDirectory() as directory:
            en = Path(directory) / 'topic_en.md'
            ja = Path(directory) / 'topic_ja.md'
            en.write_text('# Title\n## One\n### Child\n## Two\n')
            for content, expected in (
                ('# Title\n## One\n### Child\n## Two\n', 0),
                ('# Title\n## One\n## Two\n### Child\n', 1),
            ):
                ja.write_text(content)
                with self.subTest(content=content), \
                     patch.object(headings, 'find_bilingual_pairs', return_value=[(en, ja)]), \
                     contextlib.redirect_stdout(io.StringIO()):
                    self.assertEqual(headings.main(), expected)
            ja.unlink()
            with patch.object(headings, 'find_bilingual_pairs', return_value=[(en, ja)]), \
                 contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(headings.main(), 1)


class FormatterTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        source = Path(__file__).resolve().parents[1]
        shutil.copytree(source / 'tools', self.root / 'tools', ignore=shutil.ignore_patterns('__pycache__'))
        shutil.copy(source / 'Makefile', self.root)
        self.bin = self.root / 'node_modules/.bin'
        self.bin.mkdir(parents=True)

    def tool(self, name, body):
        path = self.bin / name
        path.write_text('#!/bin/sh\n' + body)
        path.chmod(0o755)

    def run_format(self):
        return subprocess.run(['make', 'format'], cwd=self.root, capture_output=True, text=True)

    def test_missing_dependency_fails_before_editing(self):
        self.tool('prettier', 'touch unexpected\n')
        result = self.run_format()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Missing markdownlint', result.stderr)
        self.assertFalse((self.root / 'unexpected').exists())

    def test_tool_failure_propagates_and_stops_pipeline(self):
        (self.root / 'sample.md').touch()
        self.tool('prettier', 'exit 7\n')
        for name in ('markdownlint', 'textlint'):
            self.tool(name, 'touch unexpected\n')
        self.assertNotEqual(self.run_format().returncode, 0)
        self.assertFalse((self.root / 'unexpected').exists())

    def test_shared_selection_preserves_spaces_and_exclusions(self):
        for file in ('a file.md', 'CLAUDE.md', 'node_modules/skip.md',
                     'nested/.gomodcache/skip.md', 'nested/CLAUDE.md'):
            path = self.root / file
            path.parent.mkdir(parents=True, exist_ok=True)
            path.touch()
        for name in ('prettier', 'markdownlint', 'textlint'):
            self.tool(name, f'printf "%s\\n" "$@" > {name}.args\n')
        result = self.run_format()
        self.assertEqual(result.returncode, 0, result.stderr)
        for name in ('prettier', 'markdownlint', 'textlint'):
            args = (self.root / f'{name}.args').read_text().splitlines()
            self.assertEqual(args[args.index('--') + 1:], ['a file.md'])


if __name__ == '__main__':
    unittest.main()
