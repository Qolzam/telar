package email

import "context"

// Message represents an email to be sent.
type Message struct {
	From    string
	To      []string
	Subject string
	Body    string // HTML allowed
}

// Sender abstracts email sending for DI and testing.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}
