// Copyright (c) 2025 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestLoadFromMap tests configuration loading from an in-memory map.
// This test is 100% parallel-safe and has no side effects.
func TestLoadFromMap(t *testing.T) {
	t.Parallel() // Now safe to run in parallel!

	t.Run("Loads all provided values correctly", func(t *testing.T) {
		t.Parallel()

		// 1. Define the entire test environment as a simple map.
		testEnv := map[string]string{
			"HMAC_SECRET":     "test-secret",
			"JWT_PRIVATE_KEY": "test-private-key",
			"JWT_PUBLIC_KEY":  "test-public-key",
			"DB_TYPE":         "mongo",
			"MONGO_URI":       "mongodb://test-mongo-host:27017/test-db",
			"SERVER_PORT":     "9090",
			"DEBUG":           "true",
			"CACHE_TTL":       "30m",
		}

		// 2. Call the new, pure function with the test environment.
		cfg, err := LoadFromMap(testEnv)
		require.NoError(t, err)

		// 3. Assert on the results.
		require.Equal(t, "test-secret", cfg.HMAC.Secret)
		require.Equal(t, "test-private-key", cfg.JWT.PrivateKey)
		require.Equal(t, "test-public-key", cfg.JWT.PublicKey)
		require.Equal(t, "mongo", cfg.Database.Type)
		require.Equal(t, 9090, cfg.Server.Port)
		require.True(t, cfg.Server.Debug)
		require.Equal(t, 30*time.Minute, cfg.Cache.TTL)
	})

	t.Run("Applies defaults for missing values", func(t *testing.T) {
		t.Parallel()

		// Provide only the bare minimum required values.
		testEnv := map[string]string{
			"HMAC_SECRET":     "test-secret",
			"JWT_PRIVATE_KEY": "test-private-key",
			"JWT_PUBLIC_KEY":  "test-public-key",
			"MONGO_URI":       "mongodb://localhost:27017",
		}

		cfg, err := LoadFromMap(testEnv)
		require.NoError(t, err)

		// Assert that defaults were applied correctly.
		require.Equal(t, 8080, cfg.Server.Port)
		require.False(t, cfg.Server.Debug)
		require.Equal(t, 1*time.Hour, cfg.Cache.TTL)
	})

	t.Run("Returns error for missing required values", func(t *testing.T) {
		t.Parallel()

		// Missing MONGO_URI
		testEnv := map[string]string{
			"HMAC_SECRET":     "test-secret",
			"JWT_PRIVATE_KEY": "test-private-key",
		}

		_, err := LoadFromMap(testEnv)
		require.Error(t, err)
		require.Contains(t, err.Error(), "MONGO_URI is not set")
	})

	t.Run("Returns error for missing JWT_PRIVATE_KEY", func(t *testing.T) {
		t.Parallel()

		testEnv := map[string]string{
			"HMAC_SECRET":    "test-secret",
			"JWT_PUBLIC_KEY": "test-public-key",
			"MONGO_URI":      "mongodb://localhost:27017",
		}

		_, err := LoadFromMap(testEnv)
		require.Error(t, err)
		require.Contains(t, err.Error(), "JWT_PRIVATE_KEY is not set")
	})

	t.Run("Returns error for missing JWT_PUBLIC_KEY", func(t *testing.T) {
		t.Parallel()

		testEnv := map[string]string{
			"HMAC_SECRET":     "test-secret",
			"JWT_PRIVATE_KEY": "test-private-key",
			"MONGO_URI":       "mongodb://localhost:27017",
		}

		_, err := LoadFromMap(testEnv)
		require.Error(t, err)
		require.Contains(t, err.Error(), "JWT_PUBLIC_KEY is not set")
	})

	t.Run("Returns error for missing HMAC_SECRET", func(t *testing.T) {
		t.Parallel()

		testEnv := map[string]string{
			"JWT_PRIVATE_KEY": "test-private-key",
			"JWT_PUBLIC_KEY":  "test-public-key",
			"MONGO_URI":       "mongodb://localhost:27017",
		}

		_, err := LoadFromMap(testEnv)
		require.Error(t, err)
		require.Contains(t, err.Error(), "HMAC_SECRET is not set")
	})

	t.Run("Handles integer parsing errors gracefully", func(t *testing.T) {
		t.Parallel()

		testEnv := map[string]string{
			"HMAC_SECRET":     "test-secret",
			"JWT_PRIVATE_KEY": "test-private-key",
			"JWT_PUBLIC_KEY":  "test-public-key",
			"MONGO_URI":       "mongodb://localhost:27017",
			"SERVER_PORT":     "not-a-number",
		}

		cfg, err := LoadFromMap(testEnv)
		require.NoError(t, err)
		// Should fall back to default port
		require.Equal(t, 8080, cfg.Server.Port)
	})

	t.Run("Handles boolean parsing errors gracefully", func(t *testing.T) {
		t.Parallel()

		testEnv := map[string]string{
			"HMAC_SECRET":     "test-secret",
			"JWT_PRIVATE_KEY": "test-private-key",
			"JWT_PUBLIC_KEY":  "test-public-key",
			"MONGO_URI":       "mongodb://localhost:27017",
			"DEBUG":           "not-a-boolean",
		}

		cfg, err := LoadFromMap(testEnv)
		require.NoError(t, err)
		// Should fall back to default false
		require.False(t, cfg.Server.Debug)
	})

	t.Run("Handles duration parsing errors gracefully", func(t *testing.T) {
		t.Parallel()

		testEnv := map[string]string{
			"HMAC_SECRET":     "test-secret",
			"JWT_PRIVATE_KEY": "test-private-key",
			"JWT_PUBLIC_KEY":  "test-public-key",
			"MONGO_URI":       "mongodb://localhost:27017",
			"CACHE_TTL":       "not-a-duration",
		}

		cfg, err := LoadFromMap(testEnv)
		require.NoError(t, err)
		// Should fall back to default duration
		require.Equal(t, 1*time.Hour, cfg.Cache.TTL)
	})

	t.Run("Handles fallback environment variables", func(t *testing.T) {
		t.Parallel()

		testEnv := map[string]string{
			"HMAC_SECRET":     "test-secret",
			"JWT_PRIVATE_KEY": "test-private-key",
			"JWT_PUBLIC_KEY":  "test-public-key",
			"MONGO_URI":       "mongodb://localhost:27017",
			"MONGO_HOST":      "fallback-host", // Should use MONGO_HOST as fallback for MONGODB_HOST
		}

		cfg, err := LoadFromMap(testEnv)
		require.NoError(t, err)
		require.Equal(t, "fallback-host", cfg.Database.MongoDB.Host)
	})
}

// TestLoadFromEnv tests the original LoadFromEnv function to ensure backward compatibility
func TestLoadFromEnv(t *testing.T) {
	// This test is intentionally NOT parallel to avoid conflicts with environment variables
	// It tests the actual LoadFromEnv function which reads from the real environment

	// Note: This test will only work if the required environment variables are set
	// In CI/CD, these should be set by the environment
	// For local development, they should be in the .env file

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Skipf("LoadFromEnv test skipped: %v (this is expected if environment variables are not set)", err)
		return
	}

	// Basic validation that the config was loaded
	require.NotNil(t, cfg)
	require.NotEmpty(t, cfg.HMAC.Secret)
	require.NotEmpty(t, cfg.JWT.PrivateKey)
	require.NotEmpty(t, cfg.JWT.PublicKey)
	require.NotEmpty(t, cfg.Database.MongoDB.URI)
}
