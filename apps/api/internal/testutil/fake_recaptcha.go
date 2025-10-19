package testutil

import (
	"context"
	"fmt"
)

// FakeRecaptchaVerifier is a test-only implementation of the recaptcha.Verifier interface.
type FakeRecaptchaVerifier struct {
	// ShouldSucceed controls the verification outcome.
	ShouldSucceed bool
	// ExpectedToken can be used to assert that a specific token was passed.
	ExpectedToken string
}

// Verify implements the recaptcha.Verifier interface for tests.
func (f *FakeRecaptchaVerifier) Verify(ctx context.Context, token string) (bool, error) {
	if f.ExpectedToken != "" && f.ExpectedToken != token {
		return false, fmt.Errorf("received unexpected recaptcha token. Got '%s', want '%s'", token, f.ExpectedToken)
	}
	if f.ShouldSucceed {
		return true, nil
	}
	return false, nil
}
