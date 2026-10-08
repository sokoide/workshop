import contextlib
import io
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'tools'))

import check_glossary_anchors as glossary


class GlossaryAnchorTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        (self.root / 'glossary_en.md').write_text(
            '# Glossary\n'
            '## <a name="network"></a>Networking Terms\n'
            '### Rate Limiting Terms\n'
            '## <a name="mq"></a><a name="messaging"></a>Message Queue Terms\n',
            encoding='utf-8')
        (self.root / 'glossary_ja.md').write_text(
            '# 用語集\n'
            '## <a name="network"></a>ネットワーク用語\n'
            '### キャッシュ制御\n',
            encoding='utf-8')

    def write(self, name: str, content: str) -> Path:
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding='utf-8')
        return path

    def test_valid_anchors_pass(self):
        source = self.write('topic_en.md',
                            '[Networking](glossary_en.md#network) '
                            '[slug](glossary_en.md#rate-limiting-terms) '
                            '[second tag](glossary_en.md#messaging) '
                            '[ja tag](glossary_ja.md#network) '
                            '[ja slug](glossary_ja.md#キャッシュ制御)\n')
        self.assertEqual(glossary.anchor_violations(source, {}), [])

    def test_unknown_anchors_are_reported(self):
        source = self.write('topic_ja.md',
                            '[bogus](glossary_en.md#container) [gone](glossary_ja.md#mq)\n')
        self.assertEqual(glossary.anchor_violations(source, {}), [
            (1, 'glossary_en.md#container', 'unknown anchor'),
            (1, 'glossary_ja.md#mq', 'unknown anchor'),
        ])

    def test_missing_glossary_file_is_reported(self):
        source = self.write('nested/topic_en.md', '[terms](../glossary_en.md#network) '
                                                  '[ja](glossary_ja.md#network)\n')
        self.assertEqual(glossary.anchor_violations(source, {}), [
            (1, 'glossary_ja.md#network', 'glossary file not found'),
        ])

    def test_headings_inside_code_fences_do_not_count(self):
        (self.root / 'glossary_en.md').write_text(
            '# Glossary\n```text\n## Fenced Heading\n```\n', encoding='utf-8')
        source = self.write('topic.md', '[fenced](glossary_en.md#fenced-heading) '
                                        '[real](glossary_en.md#glossary)\n')
        self.assertEqual(glossary.anchor_violations(source, {}), [
            (1, 'glossary_en.md#fenced-heading', 'unknown anchor'),
        ])

    def test_cli_exit_status(self):
        good = self.write('good_en.md', '[ok](glossary_en.md#network)\n')
        bad = self.write('bad_ja.md', '[ng](glossary_ja.md#container)\n')
        for files, expected in (([good], 0), ([good, bad], 1)):
            with self.subTest(files=files), contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(glossary.report(files, self.root), expected)


if __name__ == '__main__':
    unittest.main()
