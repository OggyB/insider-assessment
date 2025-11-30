package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/oggyb/insider-assessment/internal/request"
	"github.com/oggyb/insider-assessment/internal/response"
	"io"
	"net/http"
	"time"
)

var _ Client = (*WebhookClient)(nil)

// WebhookClient is an SMS client that sends messages to a webhook-style HTTP endpoint.
type WebhookClient struct {
	endpoint   string
	authKey    string
	httpClient *http.Client
}

// NewWebhookClient creates a new WebhookClient with the given endpoint and auth key.
func NewWebhookClient(endpoint, authKey string) *WebhookClient {
	return &WebhookClient{
		endpoint: endpoint,
		authKey:  authKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // ekstra güvenlik, yine de ctx ile de sınırlarız
		},
	}
}

// withTimeout wraps the context with a timeout if it doesn't already have one.
func withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		// Already has a deadline; no need to wrap again.
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

// Send implements Client.Send by posting a JSON payload to the configured webhook endpoint.
func (c *WebhookClient) Send(ctx context.Context, to, content string) (string, string, error) {
	// Keep individual requests bounded in time.
	ctx, cancel := withTimeout(ctx, 5*time.Second)
	defer cancel()

	payload := request.WebhookRequest{
		To:      to,
		Content: content,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.authKey != "" {
		req.Header.Set("x-ins-auth-key", c.authKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// context timeout / cancel ise bunu özellikle belirtelim
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return "", "", fmt.Errorf("webhook request timeout or canceled: %w", err)
		}
		return "", "", fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	rawBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read webhook response: %w", err)
	}
	raw := string(rawBytes)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", raw, fmt.Errorf("webhook returned non-2xx status: %d", resp.StatusCode)
	}

	var parsed response.WebhookResponse
	if err := json.Unmarshal(rawBytes, &parsed); err != nil {
		return "", raw, fmt.Errorf("failed to parse webhook response: %w", err)
	}

	if parsed.MessageID == "" {
		return "", raw, fmt.Errorf("webhook response missing messageId")
	}

	return parsed.MessageID, raw, nil
}

// Health implements Client.Health with a simple GET request to the webhook endpoint.
func (c *WebhookClient) Health(ctx context.Context) error {
	// Lightweight ping with a short timeout.
	ctx, cancel := withTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return fmt.Errorf("health: failed to create request: %w", err)
	}

	if c.authKey != "" {
		req.Header.Set("x-ins-auth-key", c.authKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return fmt.Errorf("health: request timeout or canceled: %w", err)
		}
		return fmt.Errorf("health: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("health: non-2xx status: %d", resp.StatusCode)
	}

	return nil
}

// compile-time check: WebhookClient satisfies the Client interface.
var _ Client = (*WebhookClient)(nil)
