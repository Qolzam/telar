// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package utils

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
)

// GenerateTransactionID generates a unique transaction ID
func GenerateTransactionID() string {
	id, err := uuid.NewV4()
	if err != nil {
		// Fallback to timestamp-based ID if UUID generation fails
		return fmt.Sprintf("tx_%d_%x", time.Now().UnixNano(), generateRandomBytes(4))
	}
	return fmt.Sprintf("tx_%s", id.String())
}

// generateRandomBytes generates random bytes for fallback ID generation
func generateRandomBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

// DefaultTransactionConfig returns a default transaction configuration
func DefaultTransactionConfig() *interfaces.TransactionConfig {
	return &interfaces.TransactionConfig{
		Timeout:        30 * time.Second,
		ReadOnly:       false,
		IsolationLevel: interfaces.IsolationLevelDefault,
		RetryPolicy: &interfaces.RetryPolicy{
			MaxRetries:      3,
			InitialDelay:    100 * time.Millisecond,
			MaxDelay:        5 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"TRANSACTION_CONFLICT", "DEADLOCK", "CONNECTION_FAILED"},
		},
	}
}

// MergeTransactionConfig merges user config with defaults
func MergeTransactionConfig(userConfig *interfaces.TransactionConfig) *interfaces.TransactionConfig {
	config := DefaultTransactionConfig()
	
	if userConfig == nil {
		return config
	}
	
	if userConfig.Timeout > 0 {
		config.Timeout = userConfig.Timeout
	}
	
	config.ReadOnly = userConfig.ReadOnly
	
	if userConfig.IsolationLevel != interfaces.IsolationLevelDefault {
		config.IsolationLevel = userConfig.IsolationLevel
	}
	
	if userConfig.RetryPolicy != nil {
		if userConfig.RetryPolicy.MaxRetries >= 0 {
			config.RetryPolicy.MaxRetries = userConfig.RetryPolicy.MaxRetries
		}
		if userConfig.RetryPolicy.InitialDelay > 0 {
			config.RetryPolicy.InitialDelay = userConfig.RetryPolicy.InitialDelay
		}
		if userConfig.RetryPolicy.MaxDelay > 0 {
			config.RetryPolicy.MaxDelay = userConfig.RetryPolicy.MaxDelay
		}
		if userConfig.RetryPolicy.BackoffFactor > 0 {
			config.RetryPolicy.BackoffFactor = userConfig.RetryPolicy.BackoffFactor
		}
		if len(userConfig.RetryPolicy.RetryableErrors) > 0 {
			config.RetryPolicy.RetryableErrors = userConfig.RetryPolicy.RetryableErrors
		}
	}
	
	return config
}

// CreateTimeoutContext creates a context with timeout based on transaction config
func CreateTimeoutContext(ctx context.Context, config *interfaces.TransactionConfig) (context.Context, context.CancelFunc) {
	if config != nil && config.Timeout > 0 {
		return context.WithTimeout(ctx, config.Timeout)
	}
	return context.WithTimeout(ctx, 30*time.Second) // Default timeout
}

// ConvertIsolationLevel converts interface isolation level to sql.IsolationLevel
func ConvertIsolationLevel(level interfaces.IsolationLevel) sql.IsolationLevel {
	switch level {
	case interfaces.IsolationLevelReadUncommitted:
		return sql.LevelReadUncommitted
	case interfaces.IsolationLevelReadCommitted:
		return sql.LevelReadCommitted
	case interfaces.IsolationLevelRepeatableRead:
		return sql.LevelRepeatableRead
	case interfaces.IsolationLevelSerializable:
		return sql.LevelSerializable
	default:
		return sql.LevelDefault
	}
}

// IsRetryableError checks if an error is retryable based on the retry policy
func IsRetryableError(err error, retryPolicy *interfaces.RetryPolicy) bool {
	if retryPolicy == nil || len(retryPolicy.RetryableErrors) == 0 {
		return false
	}
	
	errorCode := ""
	if repoErr, ok := err.(*interfaces.RepositoryError); ok {
		errorCode = repoErr.Code
	} else {
		errorCode = err.Error()
	}
	
	for _, retryableCode := range retryPolicy.RetryableErrors {
		if errorCode == retryableCode {
			return true
		}
	}
	
	return false
}

// CalculateBackoffDelay calculates the delay for the next retry attempt
func CalculateBackoffDelay(attempt int, retryPolicy *interfaces.RetryPolicy) time.Duration {
	if retryPolicy == nil {
		return time.Second
	}
	
	delay := retryPolicy.InitialDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * retryPolicy.BackoffFactor)
		if delay > retryPolicy.MaxDelay {
			delay = retryPolicy.MaxDelay
			break
		}
	}
	
	return delay
}

// ValidateTransactionConfig validates transaction configuration
func ValidateTransactionConfig(config *interfaces.TransactionConfig) error {
	if config == nil {
		return nil
	}
	
	if config.Timeout < 0 {
		return fmt.Errorf("transaction timeout cannot be negative")
	}
	
	if config.Timeout > 10*time.Minute {
		return fmt.Errorf("transaction timeout cannot exceed 10 minutes")
	}
	
	if config.RetryPolicy != nil {
		if config.RetryPolicy.MaxRetries < 0 {
			return fmt.Errorf("max retries cannot be negative")
		}
		
		if config.RetryPolicy.MaxRetries > 10 {
			return fmt.Errorf("max retries cannot exceed 10")
		}
		
		if config.RetryPolicy.InitialDelay < 0 {
			return fmt.Errorf("initial delay cannot be negative")
		}
		
		if config.RetryPolicy.MaxDelay < config.RetryPolicy.InitialDelay {
			return fmt.Errorf("max delay cannot be less than initial delay")
		}
		
		if config.RetryPolicy.BackoffFactor <= 0 {
			return fmt.Errorf("backoff factor must be positive")
		}
	}
	
	return nil
}

// ExecuteWithRetry executes a function with retry logic based on the retry policy
func ExecuteWithRetry(ctx context.Context, retryPolicy *interfaces.RetryPolicy, fn func() error) error {
	if retryPolicy == nil || retryPolicy.MaxRetries == 0 {
		return fn()
	}
	
	var lastErr error
	for attempt := 0; attempt <= retryPolicy.MaxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		
		lastErr = err
		
		// Don't retry if this is the last attempt or if the error is not retryable
		if attempt == retryPolicy.MaxRetries || !IsRetryableError(err, retryPolicy) {
			break
		}
		
		// Calculate delay and wait
		delay := CalculateBackoffDelay(attempt, retryPolicy)
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}
	
	return lastErr
}
