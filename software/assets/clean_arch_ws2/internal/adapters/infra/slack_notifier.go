package infra

import (
	"context"
	"fmt"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/usecase/port"
)

// SlackNotifier is a stand-in for the Slack API: it prints the message
// instead of calling a webhook, so the workshop runs with zero external
// dependencies.
type SlackNotifier struct {
	webhookURL string
}

func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{webhookURL: webhookURL}
}

// Compile-time check that SlackNotifier satisfies the same port.
var _ port.NotificationGateway = (*SlackNotifier)(nil)

func (n *SlackNotifier) Send(ctx context.Context, message string) error {
	fmt.Printf("[slack webhook %s] %s\n", n.webhookURL, message)
	return nil
}
