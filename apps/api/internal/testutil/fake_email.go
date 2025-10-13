package testutil

import (
	"context"
	"sync"

	platformemail "github.com/qolzam/telar/apps/api/internal/platform/email"
)

// FakeEmailSender captures emails in memory for tests.
type FakeEmailSender struct {
	mu   sync.Mutex
	Sent []platformemail.Message
}

func NewFakeEmailSender() *FakeEmailSender {
	return &FakeEmailSender{Sent: make([]platformemail.Message, 0)}
}

func (f *FakeEmailSender) Send(ctx context.Context, msg platformemail.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Sent = append(f.Sent, msg)
	return nil
}

func (f *FakeEmailSender) LastSent() *platformemail.Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.Sent) == 0 {
		return nil
	}
	return &f.Sent[len(f.Sent)-1]
}

func (f *FakeEmailSender) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Sent = make([]platformemail.Message, 0)
}
