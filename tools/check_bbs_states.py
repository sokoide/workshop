"""Build and test all maintained BBS start and reference states in temporary copies."""

from pathlib import Path
import os
import subprocess
import tempfile

from prepare_bbs import prepare


def main() -> int:
    with tempfile.TemporaryDirectory(prefix='bbs-states-') as directory:
        for workshop in ('ws3', 'ws4', 'ws5', 'ws6', 'ws7'):
            for solution in ((False, True) if workshop in ('ws3', 'ws4', 'ws5') else (False,)):
                label = workshop + ('-solution' if solution else '-start')
                destination = Path(directory) / label
                prepare(workshop, destination, solution)
                for command in (
                    ['go', 'test', '-mod=readonly', '-race', './...'],
                    ['go', 'build', '-mod=readonly', '-o', os.devnull, './...'],
                ):
                    result = subprocess.run(command, cwd=destination, capture_output=True, text=True)
                    if result.returncode:
                        print(label, result.stdout, result.stderr)
                        return result.returncode
                print(f'{label}: test/build passed', flush=True)
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
