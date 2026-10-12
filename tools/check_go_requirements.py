"""Check workshop header Go minima against their executable asset modules."""

from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]
ASSETS = {
    'software/clean_arch_ws8': 'software/assets/greeting',
    'software/clean_arch_ws1': 'software/assets/clean_arch_ws1',
    'software/clean_arch_ws2': 'software/assets/clean_arch_ws2',
    **{f'software/clean_arch_ws{n}': 'software/assets/bbs' for n in range(3, 8)},
    'software/go_cache_patterns': 'software/assets/go_cache_patterns',
    'software/go_mq_patterns': 'software/assets/go_mq_patterns',
    'software/idempotency': 'software/assets/idempotency',
    'infra/cache': 'infra/assets/redis_leaderboard',
    'infra/rate_limiting': 'infra/assets/rate_limiting',
    'infra/rabbitmq': 'infra/assets/rabbitmq_crypto',
    'infra/minio': 'infra/assets/minio',
    'infra/secret_management': 'infra/assets/secret_management',
    'infra/tcpip_stack': 'infra/assets/tcpip_stack',
    'infra/tcpip_stack_udp': 'infra/assets/tcpip_stack',
    'infra/http_persistent': 'infra/assets/http_persistent_conn',
    'infra/oauth2': 'infra/assets/oauth2',
    'infra/oauth2_entra': 'infra/assets/oauth2_entra',
    'infra/web_security': 'infra/assets/web_security',
    'infra/web_security_defense': 'infra/assets/web_security',
    'infra/supply_chain_security': 'infra/assets/supply_chain',
}
VERSION = r'(\d+\.\d+(?:\.\d+)?)'


def version_tuple(version: str) -> tuple[int, int, int]:
    parts = [int(part) for part in version.split('.')]
    return tuple((parts + [0, 0])[:3])


def minimum(asset: Path) -> str:
    versions = []
    for module in sorted(asset.rglob('go.mod')):
        match = re.search(r'^go\s+' + VERSION + r'\s*$', module.read_text(), re.M)
        if not match:
            raise ValueError(f'{module}: missing Go directive')
        versions.append(match.group(1))
    if not versions:
        raise ValueError(f'{asset}: no go.mod found')
    return max(versions, key=version_tuple)


def requirement_error(content: str, required: str) -> str | None:
    header = re.split(r'^##[ \t]+', content, maxsplit=1, flags=re.M)[0]
    match = re.search(r'\bGo\s+' + VERSION, header)
    if not match:
        return f'header must specify Go {required} or newer'
    if version_tuple(match.group(1)) < version_tuple(required):
        return f'header says Go {match.group(1)}; asset requires {required}'
    return None


def main() -> int:
    violations = []
    for stem, asset in ASSETS.items():
        try:
            required = minimum(ROOT / asset)
            for lang in ('en', 'ja'):
                path = ROOT / f'{stem}_{lang}.md'
                error = requirement_error(path.read_text(), required)
                if error:
                    violations.append(f'{path.relative_to(ROOT)}: {error}')
        except (OSError, ValueError) as error:
            violations.append(str(error))
    for violation in violations:
        print(violation)
    print(f'Go requirement violations: {len(violations)}')
    return int(bool(violations))


if __name__ == '__main__':
    raise SystemExit(main())
