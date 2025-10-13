package recaptcha

import "context"

// Verifier is the interface that wraps the basic Recaptcha verification method.
type Verifier interface {
	// Verify takes a Recaptcha token and returns true if it's valid, along with any error.
	Verify(ctx context.Context, token string) (bool, error)
}


