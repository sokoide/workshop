"""Validate every standalone Go workshop and report skipped integration tests."""

import argparse
from concurrent.futures import ThreadPoolExecutor
import json
import os
from pathlib import Path
import subprocess
import sys


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--jobs', type=int, default=3)
    parser.add_argument('--require-integration', action='store_true',
                        help='Fail if any test is skipped or Linux-only packages are unavailable.')
    args = parser.parse_args()
    if args.jobs < 1:
        parser.error('--jobs must be positive')
    root = Path(__file__).resolve().parents[1]
    modules = sorted(path.parent for area in ('infra', 'software')
                     for path in (root / area / 'assets').rglob('go.mod'))
    logs = root / '.cache' / 'go-validation'
    logs.mkdir(parents=True, exist_ok=True)
    env = os.environ.copy()
    for variable, directory in (('GOCACHE', '.gocache'), ('GOMODCACHE', '.gomodcache'),
                                ('GOPATH', '.gopath')):
        env.setdefault(variable, str(root / directory))
    host = subprocess.check_output(['go', 'env', 'GOOS'], env=env, text=True).strip()

    def check(module):
        name = str(module.relative_to(root))
        result = {'module': name, 'checks': {}, 'skipped': [], 'no_test_packages': [], 'limitations': []}
        if name == 'infra/assets/tcpip_stack' and host != 'linux':
            result['limitations'].append('Raw socket adapter and commands require Linux + CGO; only packet packages checked.')
        commands = (
            ['go', 'test', '-mod=readonly', '-json', '-race', '-count=1', '-timeout=90s', './...'],
            ['go', 'vet', '-mod=readonly', './...'],
            ['go', 'build', '-mod=readonly', '-o', os.devnull, './...'],
        )
        for command in commands:
            label = command[1]
            log = logs / (name.replace('/', '_') + '_' + label + '.log')
            try:
                # Discard build outputs so validation cannot overwrite checked-in
                # executables; this also works for modules containing only libraries.
                completed = subprocess.run(command, cwd=module, env=env, text=True,
                                           stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                           timeout=300)
                output, code = completed.stdout, completed.returncode
            except subprocess.TimeoutExpired:
                output, code = 'Command exceeded 300 seconds.\n', 1
            log.write_text(output)
            result['checks'][label] = code
            if label == 'test':
                for line in output.splitlines():
                    try:
                        event = json.loads(line)
                    except json.JSONDecodeError:
                        continue
                    if event.get('Action') == 'skip':
                        if event.get('Test'):
                            result['skipped'].append(event.get('Package', '') + '/' + event['Test'])
                        else:
                            result['no_test_packages'].append(event.get('Package', ''))
        summary = ' '.join(f'{key}={code}' for key, code in result['checks'].items())
        print(f"{name}: {summary} skipped={len(result['skipped'])}", flush=True)
        for limitation in result['limitations']:
            print(f'  LIMIT: {limitation}', flush=True)
        return result

    with ThreadPoolExecutor(max_workers=args.jobs) as pool:
        results = list(pool.map(check, modules))
    (logs / 'summary.json').write_text(json.dumps(results, indent=2) + '\n')
    failed = sum(any(result['checks'].values()) for result in results)
    incomplete = sum(bool(result['skipped'] or result['limitations']) for result in results)
    print(f'Modules: {len(results)}, failed: {failed}, with skipped/limited checks: {incomplete}')
    print(f'Logs: {logs}')
    return int(bool(failed or (args.require_integration and incomplete)))


if __name__ == '__main__':
    sys.exit(main())
