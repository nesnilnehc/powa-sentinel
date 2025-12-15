// Package notifier provides notification channel implementations.
package notifier

import (
	"context"

	"github.com/powa-team/powa-sentinel/internal/model"
)

// Notifier is the interface for sending alerts to external channels.
type Notifier interface {
	// Send sends the alert to the notification channel.
	Send(ctx context.Context, alert *model.AlertContext) error

	// Name returns the name of the notifier.
	Name() string
}
