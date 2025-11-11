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
	t.Parallel()

	t.Run("Loads all provided values correctly", func(t *testing.T) {
		t.Parallel()

		testEnv := map[string]string{
			"HMAC_SECRET":           "test-secret",
			"JWT_PRIVATE_KEY":       "test-private-key",
			"JWT_PUBLIC_KEY":        "test-public-key",
			"DB_TYPE":               "postgresql",
			"POSTGRES_DSN":          "postgres://test-user:test-pass@test-host:5433/test-db?sslmode=disable&search_path=custom",
			"POSTGRES_HOST":         "test-host",
			"POSTGRES_PORT":         "5433",
			"POSTGRES_USERNAME":     "test-user",
			"POSTGRES_PASSWORD":     "test-pass",
			"POSTGRES_DATABASE":     "test-db",
			"POSTGRES_SCHEMA":       "custom",
			"POSTGRES_MAX_OPEN_CONNS": "55",
			"POSTGRES_MAX_IDLE_CONNS": "23",
			"POSTGRES_CONN_MAX_LIFETIME": "321",
			"SERVER_PORT":           "9090",
			"DEBUG":                 "true",
			"CACHE_TTL":             "30m",
		}

		cfg, err := LoadFromMap(testEnv)
		require.NoError(t, err)

		require.Equal(t, "test-secret", cfg.HMAC.Secret)
		require.Equal(t, "test-private-key", cfg.JWT.PrivateKey)
		require.Equal(t, "test-public-key", cfg.JWT.PublicKey)
		require.Equal(t, "postgresql", cfg.Database.Type)
		require.Equal(t, "test-host", cfg.Database.Postgres.Host)
		require.Equal(t, 5433, cfg.Database.Postgres.Port)
		require.Equal(t, "test-user", cfg.Database.Postgres.Username)
		require.Equal(t, "test-pass", cfg.Database.Postgres.Password)
		require.Equal(t, "test-db", cfg.Database.Postgres.Database)
		require.Equal(t, "postgres://test-user:test-pass@test-host:5433/test-db?sslmode=disable&search_path=custom", cfg.Database.Postgres.DSN)
		require.Equal(t, "disable", cfg.Database.Postgres.SSLMode)
		require.Equal(t, 55, cfg.Database.Postgres.MaxOpenConns)
		require.Equal(t, 23, cfg.Database.Postgres.MaxIdleConns)
		require.Equal(t, 321*time.Second, cfg.Database.Postgres.ConnMaxLifetime)
		require.Equal(t, 9090, cfg.Server.Port)
		require.True(t, cfg.Server.Debug)
		require.Equal(t, 30*time.Minute, cfg.Cache.TTL)
	})

	t.Run("Applies defaults for missing values", func(t *testing.T) {
		t.Parallel()

		testEnv := map[string]string{
			"HMAC_SECRET":     "test-secret",
			"JWT_PRIVATE_KEY": "test-private-key",
			"JWT_PUBLIC_KEY":  "test-public-key",
		}

		cfg, err := LoadFromMap(testEnv)
		require.NoError(t, err)

		require.Equal(t, 8080, cfg.Server.Port)
		require.False(t, cfg.Server.Debug)
		require.Equal(t, 1*time.Hour, cfg.Cache.TTL)
		require.Equal(t, "postgresql", cfg.Database.Type)
		require.NotEmpty(t, cfg.Database.Postgres.DSN)
	})

	t.Run("Returns error for missing JWT_PRIVATE_KEY", func(t *testing.T) {
		t.Parallel()

		testEnv := map[string]string{
			"HMAC_SECRET":    "test-secret",
			"JWT_PUBLIC_KEY": "test-public-key",
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
			"SERVER_PORT":     "not-a-number",
		}

		cfg, err := LoadFromMap(testEnv)
		require.NoError(t, err)
		require.Equal(t, 8080, cfg.Server.Port)
	})

	t.Run("Handles boolean parsing errors gracefully", func(t *testing.T) {
		t.Parallel()

		testEnv := map[string]string{
			"HMAC_SECRET":     "test-secret",
			"JWT_PRIVATE_KEY": "test-private-key",
			"JWT_PUBLIC_KEY":  "test-public-key",
			"DEBUG":           "not-a-boolean",
		}

		cfg, err := LoadFromMap(testEnv)
		require.NoError(t, err)
		require.False(t, cfg.Server.Debug)
	})

	t.Run("Handles duration parsing errors gracefully", func(t *testing.T) {
		t.Parallel()

		testEnv := map[string]string{
			"HMAC_SECRET":     "test-secret",
			"JWT_PRIVATE_KEY": "test-private-key",
			"JWT_PUBLIC_KEY":  "test-public-key",
			"CACHE_TTL":       "not-a-duration",
		}

		cfg, err := LoadFromMap(testEnv)
		require.NoError(t, err)
		require.Equal(t, 1*time.Hour, cfg.Cache.TTL)
	})
}

// TestLoadFromEnv tests the original LoadFromEnv function to ensure backward compatibility
func TestLoadFromEnv(t *testing.T) {
	cfg, err := LoadFromEnv()
	if err != nil {
		t.Skipf("LoadFromEnv test skipped: %v (this is expected if environment variables are not set)", err)
		return
	}

	require.NotNil(t, cfg)
	require.NotEmpty(t, cfg.HMAC.Secret)
	require.NotEmpty(t, cfg.JWT.PrivateKey)
	require.NotEmpty(t, cfg.JWT.PublicKey)
	require.NotEmpty(t, cfg.Database.Postgres.DSN)
}
