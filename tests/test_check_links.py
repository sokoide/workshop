import contextlib
import io
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'tools'))

import check_links


class LinkViolationTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)

    def write(self, name: str, content: str) -> Path:
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding='utf-8')
        return path

    def test_valid_relative_links_pass(self):
        self.write('docs/peer_en.md', '# Peer\n')
        self.write('docs/diagram.png', '')
        self.write('docs/assets/data.csv', '')
        source = self.write('docs/topic_en.md',
                            '[peer](peer_en.md) ![diagram](diagram.png) '
                            '[csv](assets/data.csv) [parent](../README.md) '
                            '[page](#section) [site](https://example.com/a.md) '
                            '[mail](mailto:scott@example.com)\n')
        self.write('README.md', '# Root\n')
        self.assertEqual(check_links.link_violations(source), [])

    def test_links_inside_code_fences_are_ignored(self):
        source = self.write('topic_en.md',
                            '```text\n[example](missing_en.md) ![gone](gone.png)\n```\n')
        self.assertEqual(check_links.link_violations(source), [])

    def test_missing_targets_are_reported(self):
        source = self.write('topic_en.md', '[missing](gone_en.md) ![gone](gone.png)\n')
        self.assertEqual(check_links.link_violations(source), [
            (1, 'gone_en.md', 'missing target'),
            (1, 'gone.png', 'missing target'),
        ])

    def test_absolute_paths_are_reported(self):
        source = self.write('topic_en.md',
                            '[root](/etc/hosts) [home](nested/../../home/scott/notes.md)\n')
        self.assertEqual(check_links.link_violations(source), [
            (1, '/etc/hosts', 'absolute path'),
            (1, 'nested/../../home/scott/notes.md', 'absolute path'),
        ])

    def test_cli_exit_status(self):
        good = self.write('good_en.md', '[ok](peer.md)\n')
        self.write('peer.md', '')
        bad = self.write('bad_en.md', '[abs](/Users/scott/file.md)\n')
        for files, expected in (([good], 0), ([good, bad], 1)):
            with self.subTest(files=files), contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(check_links.report(files, self.root), expected)


if __name__ == '__main__':
    unittest.main()
