import contextlib
import io
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from tools import check_workshop_skeleton as skeleton


GOOD_EN = """# Sample Workshop

> **⏱️ Estimated Time**: Approx. 30 minutes
> **🎯 Level**: Beginner

## Goal

Do things.

## Prerequisites

- tool

## Steps

### STEP 1: Run

```bash
echo hi
# Expected output:
# hi
```

### ✅ Checkpoint

- [ ] done

## What you learned

- something

## Cleanup

```bash
rm -rf work
```

## References

- <https://example.com>
"""

# Wording synonyms observed in the corpus: ゴール/前提知識/まとめ/後片付け/参考リンク.
GOOD_JA = """# サンプル実習

> **⏱️ 所要時間**: 約 30 分
> **🎯 レベル**: 初級

## ゴール

何かをする。

## 前提知識

- ツール

## 手順

### STEP 1: 実行

```bash
echo hi
# Expected output:
# hi
```

### ✅ チェックポイント

- [ ] 確認

## まとめ

- 何かを学んだ

## 🧹 後片付け

```bash
rm -rf work
```

## 参考リンク

- <https://example.com>
"""


def stripped(template: str, needle: str) -> str:
    """Remove the first line containing needle (and its blockquote partner)."""
    lines = template.splitlines()
    kept = [line for line in lines if needle not in line]
    return '\n'.join(kept) + '\n'


class SkeletonTests(unittest.TestCase):
    def test_complete_documents_pass(self):
        self.assertEqual(skeleton.check_skeleton(GOOD_EN, 'en'), [])
        self.assertEqual(skeleton.check_skeleton(GOOD_JA, 'ja'), [])

    def test_missing_items_are_reported(self):
        cases = {
            'header ⏱️ (estimated time)': stripped(GOOD_EN, '⏱'),
            'header 🎯 (level)': stripped(GOOD_EN, '🎯'),
            'section: Goal': stripped(GOOD_EN, '## Goal'),
            'section: Prerequisites': stripped(GOOD_EN, '## Prerequisites'),
            'section: What you learned': stripped(GOOD_EN, '## What you learned'),
            'section: Cleanup': stripped(GOOD_EN, '## Cleanup'),
            'section: References': stripped(GOOD_EN, '## References'),
            'checkpoint (✅ line)': stripped(GOOD_EN, '✅ Checkpoint'),
            'expected output comment': stripped(GOOD_EN, '# Expected output:'),
        }
        for item, content in cases.items():
            with self.subTest(item=item):
                self.assertIn(item, skeleton.check_skeleton(content, 'en'))

    def test_fenced_heading_and_checkpoint_do_not_count(self):
        # `## Goal` and a ✅ line inside a fence must not satisfy their checks.
        content = stripped(stripped(GOOD_EN, '## Goal'), '✅ Checkpoint').replace(
            '# hi', '# hi\n## Goal\n### ✅ fenced checkpoint')
        missing = skeleton.check_skeleton(content, 'en')
        self.assertIn('section: Goal', missing)
        self.assertIn('checkpoint (✅ line)', missing)

    def test_language_specific_patterns(self):
        # English wording in a ja file (and vice versa) is still recognized,
        # but ja-only synonyms must not satisfy the en check.
        self.assertIn('section: Cleanup', skeleton.check_skeleton(GOOD_EN.replace('## Cleanup', '## 片付け'), 'en'))

    def test_exempt_files_are_skipped(self):
        # The skeleton-less glossary pair is exempt, so only the complete
        # topic pair is checked and the run still exits 0.
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name, content in (('glossary_en.md', '# Title\n'),
                                  ('glossary_ja.md', '# Title\n'),
                                  ('topic_en.md', GOOD_EN),
                                  ('topic_ja.md', GOOD_JA)):
                (root / name).write_text(content)
            pairs = skeleton.find_bilingual_pairs(root)
            self.assertEqual(len(pairs), 2)
            with patch.object(skeleton, 'find_bilingual_pairs', return_value=pairs), \
                 contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(skeleton.main(), 0)

    def test_pair_mismatch_is_reported_per_file(self):
        # Only the ja side lacks the Goal section; it alone is a violation.
        en_missing = skeleton.check_skeleton(GOOD_EN, 'en')
        ja_missing = skeleton.check_skeleton(stripped(GOOD_JA, '## ゴール'), 'ja')
        self.assertEqual(en_missing, [])
        self.assertIn('section: Goal', ja_missing)

    def test_cli_exit_status(self):
        with tempfile.TemporaryDirectory() as directory:
            en = Path(directory) / 'topic_en.md'
            ja = Path(directory) / 'topic_ja.md'
            en.write_text(GOOD_EN)
            for content, expected in (
                (GOOD_JA, 0),
                (stripped(GOOD_JA, '## ゴール'), 1),
            ):
                ja.write_text(content)
                with self.subTest(expected=expected), \
                     patch.object(skeleton, 'find_bilingual_pairs', return_value=[(en, ja)]), \
                     contextlib.redirect_stdout(io.StringIO()):
                    self.assertEqual(skeleton.main(), expected)


if __name__ == '__main__':
    unittest.main()
