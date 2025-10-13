package recaptcha

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	
	"github.com/qolzam/telar/apps/api/internal/types"
)

// googleRecaptchaVerifier is the production implementation of the Verifier interface.
type googleRecaptchaVerifier struct {
	secretKey  string
	httpClient *http.Client
}

// NewGoogleVerifier creates a new production-ready verifier.
func NewGoogleVerifier(secretKey string) (Verifier, error) {
	if secretKey == "" {
		return nil, fmt.Errorf("recaptcha secret key cannot be empty")
	}
	return &googleRecaptchaVerifier{
		secretKey: secretKey,
		httpClient: &http.Client{ Timeout: 5 * time.Second },
	}, nil
}

type googleResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

func (v *googleRecaptchaVerifier) Verify(ctx context.Context, token string) (bool, error) {
	formData := url.Values{
		"secret":   {v.secretKey},
		"response": {token},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://www.google.com/recaptcha/api/siteverify", strings.NewReader(formData.Encode()))
	if err != nil {
		return false, fmt.Errorf("failed to create recaptcha request: %w", err)
	}
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to call recaptcha api: %w", err)
	}
	defer resp.Body.Close()

	var googleResp googleResponse
	if err := json.NewDecoder(resp.Body).Decode(&googleResp); err != nil {
		return false, fmt.Errorf("failed to decode recaptcha response: %w", err)
	}
	return googleResp.Success, nil
}


