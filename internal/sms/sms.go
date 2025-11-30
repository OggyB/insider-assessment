// Package sms exposes a minimal interface for sending SMS messages
// and checking the health of the underlying provider.
package sms

import "context"

// Client is the contract for an SMS provider implementation.
type Client interface {
	// Send sends an SMS to the given recipient.
	// Returns an external message ID, raw provider response, and error if any.
	Send(ctx context.Context, to, content string) (externalID string, rawResponse string, err error)

	// Health checks whether the SMS provider is reachable and usable.
	Health(ctx context.Context) error
}
