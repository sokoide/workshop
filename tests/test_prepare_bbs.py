from pathlib import Path
import tempfile
import unittest

from tools.prepare_bbs import ROOT, prepare


class PrepareBBSTests(unittest.TestCase):
    def test_existing_destination_is_never_overwritten(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)
            (path / 'learner.go').write_text('my work')
            with self.assertRaises(ValueError):
                prepare('ws3', path)
            self.assertEqual((path / 'learner.go').read_text(), 'my work')

    def test_broken_destination_symlink_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory) / 'missing-target'
            link = Path(directory) / 'existing-link'
            link.symlink_to(target, target_is_directory=True)
            with self.assertRaises(ValueError):
                prepare('ws3', link)
            self.assertFalse(target.exists())
            self.assertTrue(link.is_symlink())

    def test_repository_destination_is_rejected(self):
        with self.assertRaises(ValueError):
            prepare('ws3', ROOT / 'exercise-forbidden')

    def test_states_have_correct_transport_and_scaffold(self):
        with tempfile.TemporaryDirectory() as directory:
            for workshop, solution in [('ws3', False), ('ws3', True), ('ws4', False), ('ws5', True), ('ws6', False)]:
                path = Path(directory) / (workshop + str(solution))
                prepare(workshop, path, solution)
                main = (path / 'cmd/bbs/main.go').read_text()
                self.assertEqual('grpc.NewServer' in main, workshop == 'ws3' and solution)
                self.assertEqual('postgres.OpenDB' in main, workshop in ('ws5', 'ws6'))
                self.assertTrue((path / '.baseline/internal/usecase').is_dir())
                if workshop == 'ws6':
                    self.assertFalse((path / 'cmd/bbs-postgres').exists())
                if workshop == 'ws4':
                    self.assertFalse((path / 'cmd/bbs-grpc').exists())
                    self.assertFalse((path / 'internal/adapters/infra/persistence/postgres').exists())
                    self.assertNotIn('if !thread.CanPost', (path / 'internal/usecase/post_usecase.go').read_text())
                    self.assertIn('TODO WS4', (path / 'internal/domain/entity/thread.go').read_text())


if __name__ == '__main__':
    unittest.main()
