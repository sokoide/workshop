// Package port holds the output ports owned by the UseCase layer.
package port

import "context"

// NotificationGateway is the port for sending notifications. The UseCase
// layer owns it because notification is a tool for the application workflow,
// not part of the core domain language. "How" to send (Email, Slack, ...) is
// an infrastructure detail implemented in the Infra Adapters layer.
type NotificationGateway interface {
	Send(ctx context.Context, message string) error
}
