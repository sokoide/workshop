package infra

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout は Send が標準出力へ書くメッセージ成形を検証するための
// ヘルパー。fmt.Printf の出力を一時的にパイプへ切り替える。
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}
	return string(out)
}

func TestEmailNotifier_Send(t *testing.T) {
	notifier := NewEmailNotifier("noreply@example.com")

	output := captureStdout(t, func() {
		if err := notifier.Send(context.Background(), "Membership approved: welcome, Alice!"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.HasPrefix(output, "[email from noreply@example.com] ") {
		t.Errorf("output should start with the from header, got %q", output)
	}
	if !strings.HasSuffix(output, "Membership approved: welcome, Alice!\n") {
		t.Errorf("output should end with the message, got %q", output)
	}
}

func TestSlackNotifier_Send(t *testing.T) {
	notifier := NewSlackNotifier("https://hooks.example.com/T000/B000/xxx")

	output := captureStdout(t, func() {
		if err := notifier.Send(context.Background(), "Membership approved: welcome, Alice!"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.HasPrefix(output, "[slack webhook https://hooks.example.com/T000/B000/xxx] ") {
		t.Errorf("output should start with the webhook header, got %q", output)
	}
	if !strings.HasSuffix(output, "Membership approved: welcome, Alice!\n") {
		t.Errorf("output should end with the message, got %q", output)
	}
}
