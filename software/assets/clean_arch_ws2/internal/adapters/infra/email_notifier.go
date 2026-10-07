package infra

import (
	"context"
	"fmt"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/usecase/port"
)

// EmailNotifier is a stand-in for an SMTP client: it prints the message
// instead of sending mail, so the workshop runs with zero external
// dependencies.
type EmailNotifier struct {
	from string
}

func NewEmailNotifier(from string) *EmailNotifier {
	return &EmailNotifier{from: from}
}

// Compile-time check that EmailNotifier satisfies the UseCase-owned port.
var _ port.NotificationGateway = (*EmailNotifier)(nil)

func (n *EmailNotifier) Send(ctx context.Context, message string) error {
	fmt.Printf("[email from %s] %s\n", n.from, message)
	return nil
}
