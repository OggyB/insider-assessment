package request

// SchedulerRequest represents the JSON body for scheduler control.
type SchedulerRequest struct {
	// Action controls the scheduler. Allowed values:
	// - "start": start processing batches
	// - "stop":  stop processing batches
	Action string `json:"action"`
}

type WebhookRequest struct {
	To      string `json:"to"`
	Content string `json:"content"`
}
