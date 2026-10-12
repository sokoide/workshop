from pathlib import Path
import tempfile
import unittest

from tools.check_go_requirements import minimum, requirement_error


class GoRequirementsTests(unittest.TestCase):
    def test_patch_versions_and_header_scope(self):
        self.assertIsNotNone(requirement_error('> Go 1.25+\n## Steps\nGo 1.26.2', '1.25.6'))
        self.assertIsNone(requirement_error('> Go 1.25.6+', '1.25.6'))
        self.assertIsNone(requirement_error('> Go 1.26+', '1.25.6'))
        self.assertIsNotNone(requirement_error('## Steps\nGo 1.26.2', '1.26.2'))

    def test_highest_module_minimum_is_required(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name, version in [('api', '1.26.1'), ('client', '1.25.9')]:
                (root / name).mkdir()
                (root / name / 'go.mod').write_text(f'module {name}\n\ngo {version}\n')
            self.assertEqual(minimum(root), '1.26.1')
            (root / 'api' / 'go.mod').write_text('module api\n')
            with self.assertRaises(ValueError):
                minimum(root)


if __name__ == '__main__':
    unittest.main()
