"""Create an isolated, reproducible BBS exercise without overwriting learner work."""

import argparse
from pathlib import Path
import re
import shutil

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / 'software' / 'assets' / 'bbs'


def replace_once(path: Path, before: str, after: str) -> None:
    text = path.read_text()
    if text.count(before) != 1:
        raise ValueError(f'{path}: scaffold no longer matches reference')
    path.write_text(text.replace(before, after, 1))


def prepare(workshop: str, destination: Path, solution: bool = False) -> None:
    destination = destination.resolve()
    # No merge into an existing directory, no writes within the source repository.
    if destination.exists() or destination.is_symlink():
        raise ValueError(f'destination already exists: {destination}')
    if destination == ROOT or ROOT in destination.parents:
        raise ValueError('choose an exercise directory outside the repository')
    def ignore(directory, names):
        skipped = shutil.ignore_patterns('*.db', '*.db-*', '*.log', 'bin', '.baseline')(directory, names)
        skipped.update(name for name in ('bbs', 'bbs-grpc', 'bbs-postgres')
                       if name in names and (Path(directory) / name).is_file())
        return skipped
    shutil.copytree(SOURCE, destination, ignore=ignore)
    if workshop == 'ws4' and not solution:
        replace_once(destination / 'internal/domain/entity/thread.go',
                     'return t.Owner == author',
                     '// TODO WS4: allow only the thread owner in owner-only mode.\n\treturn true')
        replace_once(destination / 'internal/usecase/post_usecase.go',
                     '\t\tif !thread.CanPost(in.Author) {\n\t\t\treturn domain.ErrNotThreadOwner\n\t\t}\n',
                     '\t\t// TODO WS4: reject a non-owner before counting or saving posts.\n')
        for relative, name in [
            ('internal/domain/entity/thread_test.go', 'TestCanPost_OwnerOnlyEnabled'),
            ('internal/usecase/post_usecase_test.go', 'TestCreatePost_OwnerOnlyRejected'),
        ]:
            path = destination / relative
            path.write_text(re.sub(r'^func '+name+r'\(.*?(?=^func |\Z)', '',
                                   path.read_text(), flags=re.M | re.S))
    if workshop == 'ws3':
        if solution:
            shutil.copyfile(destination / 'cmd/bbs-grpc/main.go', destination / 'cmd/bbs/main.go')
        else:
            shutil.rmtree(destination / 'api')
            shutil.rmtree(destination / 'internal/adapters/presentation/grpc')
            shutil.rmtree(destination / 'cmd/bbs-grpc')
    if workshop == 'ws5' and not solution:
        shutil.rmtree(destination / 'internal/adapters/infra/persistence/postgres')
        shutil.rmtree(destination / 'cmd/bbs-postgres')
    if workshop == 'ws6' or (workshop == 'ws5' and solution):
        shutil.copyfile(destination / 'cmd/bbs-postgres/main.go', destination / 'cmd/bbs/main.go')
    # Each lab includes only its transport/persistence adapters. Completed
    # adapter contract tests must not run against WS4's unfinished policy.
    if workshop != 'ws3':
        for relative in ('api', 'internal/adapters/presentation/grpc', 'cmd/bbs-grpc'):
            path = destination / relative
            if path.exists():
                shutil.rmtree(path)
    if workshop in ('ws3', 'ws4', 'ws7'):
        for relative in ('internal/adapters/infra/persistence/postgres', 'cmd/bbs-postgres'):
            path = destination / relative
            if path.exists():
                shutil.rmtree(path)
    # Hidden baseline is ignored by Go's ./... wildcard and permits diff without commits.
    baseline = destination / '.baseline'
    baseline.mkdir()
    for name in ('internal', 'cmd', 'api'):
        if (destination / name).exists():
            shutil.copytree(destination / name, baseline / name)
    for name in ('go.mod', 'go.sum'):
        shutil.copyfile(destination / name, baseline / name)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('workshop', choices=['ws3', 'ws4', 'ws5', 'ws6', 'ws7'])
    parser.add_argument('destination', type=Path)
    parser.add_argument('--solution', action='store_true',
                        help='completed reference (WS3/WS4/WS5 only)')
    args = parser.parse_args()
    if args.solution and args.workshop not in ('ws3', 'ws4', 'ws5'):
        parser.error('solutions are supplied for WS3, WS4 and WS5')
    try:
        prepare(args.workshop, args.destination, args.solution)
    except (OSError, ValueError) as error:
        parser.exit(1, f'{error}\n')
    print(f'Prepared {args.workshop}: {args.destination.absolute()}')
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
