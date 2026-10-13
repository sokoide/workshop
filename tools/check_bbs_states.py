"""Build and test all maintained BBS start and reference states in temporary copies."""

from pathlib import Path
import os
import re
import subprocess
import tempfile

from prepare_bbs import prepare



def check_ws6_recipe(root: Path, content: str, destination: Path) -> int:
    prepare('ws6', destination)

    def snippet(name: str) -> str:
        match = re.search(r'```go\n(// ' + re.escape(name) + r'.*?)```', content, re.S)
        if match is None:
            raise ValueError(f'missing WS6 example: {name}')
        return match.group(1)

    (destination / 'internal/usecase/notification.go').write_text(snippet('internal/usecase/notification.go'))
    path = destination / 'internal/usecase/post_usecase.go'
    original = path.read_text()
    change = snippet('internal/usecase/post_usecase.go')
    change = change[change.index('type CreatePostUseCase struct'):]
    path.write_text(original[:original.index('type CreatePostUseCase struct')]
                    .replace('"context"', '"context"\n"log/slog"') + change)
    for relative, package in (
        ('internal/usecase/post_usecase_test.go', 'usecase'),
        ('internal/adapters/infra/persistence/postgres/repository_test.go', 'postgres_test'),
    ):
        path = destination / relative
        path.write_text(re.sub(r'NewCreatePostUseCase\(([^()]+)\)',
                               r'NewCreatePostUseCase(\1, testNotifier{})', path.read_text()))
        (path.parent / 'test_notifier_test.go').write_text(
            'package ' + package + '\nimport "context"\n'
            'type testNotifier struct{}\n'
            'func (testNotifier) NotifyNewPost(context.Context,string,string,string) error{return nil}\n')
    # Step 2 is independently testable even before root wiring changes.
    result = subprocess.run(['go', 'test', './internal/usecase'], cwd=destination,
                            capture_output=True, text=True)
    if result.returncode:
        print(result.stdout, result.stderr)
        return result.returncode
    for name in ('noop_gateway.go', 'slack_gateway.go'):
        path = destination / 'internal/adapters/infra/notification' / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(snippet('internal/adapters/infra/notification/' + name))
    path = destination / 'cmd/bbs/main.go'
    path.write_text(path.read_text()
                   .replace('"os"', '"os"\n"github.com/sokoide/workshop/software/assets/bbs/internal/adapters/infra/notification"')
                   .replace('NewCreatePostUseCase(threadRepo, postRepo, tm)',
                            'NewCreatePostUseCase(threadRepo, postRepo, tm, notification.NewNoOpGateway())'))
    for command in (['go', 'test', '-race', './...'],
                    ['go', 'build', '-o', os.devnull, './...']):
        result = subprocess.run(command, cwd=destination, capture_output=True, text=True)
        if result.returncode:
            print(result.stdout, result.stderr)
            return result.returncode
    return 0


def check_notification_examples(directory: Path) -> int:
    """Compile both language examples and call Slack only through a local fixture."""
    root = Path(__file__).resolve().parents[1]
    version = re.search(r'^go (.+)$', (root / 'software/assets/bbs/go.mod').read_text(), re.M).group(1)
    for language in ('en', 'ja'):
        content = (root / f'software/clean_arch_ws6_{language}.md').read_text()
        destination = directory / f'notification-{language}'
        destination.mkdir()
        (destination / 'go.mod').write_text(f'module notification-example\n\ngo {version}\n')
        for name in ('noop_gateway.go', 'slack_gateway.go'):
            pattern = r'```go\n(// internal/adapters/infra/notification/' + re.escape(name) + r'.*?)```'
            match = re.search(pattern, content, re.S)
            if match is None:
                print(f'{language}: missing complete notification example {name}')
                return 1
            (destination / name).write_text(match.group(1))
        (destination / 'gateway_test.go').write_text(NOTIFICATION_TEST)
        result = subprocess.run(['go', 'test', '-race', './...'], cwd=destination,
                                capture_output=True, text=True)
        if result.returncode:
            print(result.stdout, result.stderr)
            return result.returncode
        if check_ws6_recipe(root, content, directory / f'ws6-recipe-{language}'):
            return 1
        print(f'WS6 {language} recipe and NoOp/Slack examples: test/build and local HTTP checks passed', flush=True)
    return 0


NOTIFICATION_TEST = r"""package notification

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
)
func TestNoOp(t *testing.T) {
    if err := NewNoOpGateway().NotifyNewPost(context.Background(), "thread", "author", "body"); err != nil {t.Fatal(err)}
}
func TestSlackLocalFixture(t *testing.T) {
    received := make(chan string,1)
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" {t.Error("request format")}
        var payload map[string]string
        if err:=json.NewDecoder(r.Body).Decode(&payload);err!=nil{t.Error(err)}
        received<-payload["text"]
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()
    ctx,cancel:=context.WithTimeout(context.Background(),2*time.Second);defer cancel()
    if err:=NewSlackGateway(server.URL).NotifyNewPost(ctx,"thread","author","body");err!=nil{t.Fatal(err)}
    if text:=<-received;text!="[thread] author: body"{t.Fatalf("payload=%q",text)}
}
func TestSlackFailure(t *testing.T) {
    server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.WriteHeader(http.StatusServiceUnavailable)}))
    defer server.Close()
    ctx,cancel:=context.WithTimeout(context.Background(),2*time.Second);defer cancel()
    if err:=NewSlackGateway(server.URL).NotifyNewPost(ctx,"thread","author","body");err==nil{t.Fatal("expected HTTP failure")}
}
"""


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
        return check_notification_examples(Path(directory))


if __name__ == '__main__':
    raise SystemExit(main())
