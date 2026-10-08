import contextlib
import io
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'tools'))

import check_asset_paths as assets


class AssetPathTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        (self.root / 'infra' / 'assets' / 'redis_leaderboard').mkdir(parents=True)
        (self.root / 'infra' / 'assets' / 'redis_leaderboard' / 'main.go').touch()
        (self.root / 'software' / 'assets' / 'clean_arch_ws1').mkdir(parents=True)

    def write(self, name: str, content: str) -> Path:
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding='utf-8')
        return path

    def test_existing_asset_paths_pass(self):
        source = self.write('infra/topic_en.md',
                            'Run `make -C infra/assets/redis_leaderboard up` and see '
                            '[the source](infra/assets/redis_leaderboard/main.go:42). '
                            'Then: cp -r software/assets/clean_arch_ws1 ~/work.\n')
        self.assertEqual(assets.asset_violations(source, self.root), [])

    def test_relative_asset_references_are_skipped(self):
        source = self.write('infra/topic_en.md',
                            'See [assets](assets/definitely_missing) and assets/gone/x.go.\n')
        self.assertEqual(assets.asset_violations(source, self.root), [])

    def test_missing_asset_paths_are_reported(self):
        source = self.write('README.md',
                            'Copy infra/assets/rabbitmq_crypto and '
                            'software/assets/go_cache_patterns/README.md first.\n')
        self.assertEqual(assets.asset_violations(source, self.root), [
            (1, 'infra/assets/rabbitmq_crypto'),
            (1, 'software/assets/go_cache_patterns/README.md'),
        ])

    def test_go_wildcards_and_sentence_periods_pass(self):
        (self.root / 'software' / 'assets' / 'module').mkdir(parents=True)
        source = self.write('software/topic_en.md',
                            'go test ./software/assets/module/... covers '
                            'software/assets/module. Run it.\n')
        self.assertEqual(assets.asset_violations(source, self.root), [])

    def test_cli_exit_status(self):
        good = self.write('good.md', 'cp -r software/assets/clean_arch_ws1 .\n')
        bad = self.write('bad.md', 'cp -r infra/assets/nope .\n')
        for files, expected in (([good], 0), ([good, bad], 1)):
            with self.subTest(files=files), contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(assets.report(files, self.root), expected)


if __name__ == '__main__':
    unittest.main()
