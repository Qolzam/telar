package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv" // Import the library
)

// Config represents the new, clean configuration structure
type Config struct {
	Server     ServerConfig     `json:"server"`
	Database   DatabaseConfig   `json:"database"`
	JWT        JWTConfig        `json:"jwt"`
	HMAC       HMACConfig       `json:"hmac"`
	Email      EmailConfig      `json:"email"`
	Security   SecurityConfig   `json:"security"`
	App        AppConfig        `json:"app"`
	External   ExternalConfig   `json:"external"`
	Cache      CacheConfig      `json:"cache"`
	RateLimits RateLimitsConfig `json:"rateLimits"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	BaseRoute       string `json:"baseRoute"`
	Gateway         string `json:"gateway"`
	InternalGateway string `json:"internalGateway"`
	WebDomain       string `json:"webDomain"`
	Debug           bool   `json:"debug"`
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Type                  string           `json:"type"`
	Postgres              PostgreSQLConfig `json:"postgres"`
	ForceNonTransactional bool             `json:"forceNonTransactional"`
}

// PostgreSQLConfig holds PostgreSQL-specific configuration
type PostgreSQLConfig struct {
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	Username        string        `json:"username"`
	Password        string        `json:"password"`
	Database        string        `json:"database"`
	DSN             string        `json:"dsn"`
	SSLMode         string        `json:"sslMode"`
	MaxOpenConns    int           `json:"maxOpenConns"`
	MaxIdleConns    int           `json:"maxIdleConns"`
	ConnMaxLifetime time.Duration `json:"connMaxLifetime"`
}

// JWTConfig holds JWT-related configuration
type JWTConfig struct {
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

// HMACConfig holds HMAC-related configuration
type HMACConfig struct {
	Secret string `json:"secret"`
}

// EmailConfig holds email-related configuration
type EmailConfig struct {
	SMTPEmail    string `json:"smtpEmail"`
	SMTPHost     string `json:"smtpHost"`
	SMTPPort     int    `json:"smtpPort"`
	SMTPUser     string `json:"smtpUser"`
	SMTPPass     string `json:"smtpPass"`
	RefEmail     string `json:"refEmail"`
	RefEmailPass string `json:"refEmailPass"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	RecaptchaSiteKey  string `json:"recaptchaSiteKey"`
	RecaptchaKey      string `json:"recaptchaKey"`
	RecaptchaDisabled bool   `json:"recaptchaDisabled"`
	Origin            string `json:"origin"`
}

// AppConfig holds application-related configuration
type AppConfig struct {
	WebDomain      string `json:"webDomain"`
	OrgName        string `json:"orgName"`
	Name           string `json:"name"`
	OrgAvatar      string `json:"orgAvatar"`
	QueryPrettyURL bool   `json:"queryPrettyUrl"`
}

// ExternalConfig holds external service configuration
type ExternalConfig struct {
	GitHubClientID    string `json:"githubClientId"`
	GitHubSecret      string `json:"githubSecret"`
	GoogleClientID    string `json:"googleClientId"`
	GoogleSecret      string `json:"googleSecret"`
	PhoneSourceNumber string `json:"phoneSourceNumber"`
	PhoneAuthToken    string `json:"phoneAuthToken"`
	PhoneAuthId       string `json:"phoneAuthId"`
}

// CacheConfig holds cache-related configuration
type CacheConfig struct {
	MaxMemory       int64         `json:"maxMemory"`
	TTL             time.Duration `json:"ttl"`
	Enabled         bool          `json:"enabled"`
	Backend         string        `json:"backend"`
	Prefix          string        `json:"prefix"`
	CleanupInterval time.Duration `json:"cleanupInterval"`
	Redis           RedisConfig   `json:"redis"`
}

// RateLimitConfig holds rate limiting configuration for a specific endpoint
type RateLimitConfig struct {
	Enabled  bool          `json:"enabled"`
	Max      int           `json:"max"`
	Duration time.Duration `json:"duration"`
}

// RateLimitsConfig holds rate limiting configuration for all endpoints
type RateLimitsConfig struct {
	Signup        RateLimitConfig `json:"signup"`
	Login         RateLimitConfig `json:"login"`
	PasswordReset RateLimitConfig `json:"passwordReset"`
	Verification  RateLimitConfig `json:"verification"`
}

// RedisConfig holds Redis-specific configuration
type RedisConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	Password     string        `json:"password"`
	DB           int           `json:"db"`
	PoolSize     int           `json:"poolSize"`
	MinIdle      int           `json:"minIdle"`
	Address      string        `json:"address"`
	Database     int           `json:"database"`
	MinIdleConns int           `json:"minIdleConns"`
	MaxConnAge   time.Duration `json:"maxConnAge"`
	Cluster      ClusterConfig `json:"cluster"`
}

// ClusterConfig holds Redis cluster configuration
type ClusterConfig struct {
	Addrs     []string `json:"addrs"`
	Password  string   `json:"password"`
	Enabled   bool     `json:"enabled"`
	Addresses []string `json:"addresses"`
}

// LoadFromEnv loads configuration from the environment.
// It follows a clear precedence:
// 1. Explicit Environment Variables (e.g., set in the shell or by CI)
// 2. Values from the .env file (if it exists)
// 3. Hardcoded defaults (if applicable)
func LoadFromEnv() (*Config, error) {
	// godotenv.Load() will read the .env file and load its values into the
	// environment for this process *only if they are not already set*.
	// This automatically creates the correct precedence.
	// Try multiple possible locations for .env file
	envPaths := []string{
		".env",          // Current directory
		"apps/api/.env", // From project root
		"../.env",       // From internal/platform/config
		"../../.env",    // From internal/platform/config
	}

	var loadErr error
	for _, envPath := range envPaths {
		loadErr = godotenv.Load(envPath)
		if loadErr == nil {
			break // Successfully loaded
		}
	}

	if loadErr != nil {
		// It's not an error if the .env file doesn't exist.
		// We can log a warning for clarity.
		fmt.Println("INFO: .env file not found, using environment variables and defaults.")
	}

	// The rest of the function remains the same. The os.Getenv calls will now
	// automatically see the values that were loaded from the .env file.
	config := &Config{
		Server: ServerConfig{
			Host:            getEnvOrDefault("HOST", "localhost"),
			Port:            getEnvAsInt("SERVER_PORT", 8080),
			BaseRoute:       getEnvOrDefault("BASE_ROUTE", "/api"),
			Gateway:         getEnvOrDefault("GATEWAY", "http://localhost:8080"),
			InternalGateway: getEnvOrDefault("INTERNAL_GATEWAY", "http://localhost:8080"),
			WebDomain:       getEnvOrDefault("WEB_DOMAIN", "http://localhost:3000"),
			Debug:           getEnvAsBool("DEBUG", false),
		},
		Database: DatabaseConfig{
			Type:                  getEnvOrDefault("DB_TYPE", "postgresql"),
			ForceNonTransactional: getEnvAsBool("FORCE_NON_TRANSACTIONAL", false),
			Postgres: PostgreSQLConfig{
				Host:            getEnvOrDefault("POSTGRES_HOST", "localhost"),
				Port:            getEnvAsInt("POSTGRES_PORT", 5432),
				Username:        getEnvOrDefault("POSTGRES_USERNAME", ""),
				Password:        getEnvOrDefault("POSTGRES_PASSWORD", ""),
				Database:        getEnvOrDefault("POSTGRES_DATABASE", "telar"),
				DSN:             getEnvOrDefault("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/telar_social_test?sslmode=disable&search_path=public"),
				SSLMode:         getEnvOrDefault("POSTGRES_SSL_MODE", "disable"),
				MaxOpenConns:    getEnvAsInt("POSTGRES_MAX_OPEN_CONNS", 25),
				MaxIdleConns:    getEnvAsInt("POSTGRES_MAX_IDLE_CONNS", 25),
				ConnMaxLifetime: time.Duration(getEnvAsInt("POSTGRES_CONN_MAX_LIFETIME", 300)) * time.Second,
			},
		},
		JWT: JWTConfig{
			PublicKey:  getEnvOrDefault("JWT_PUBLIC_KEY", ""),
			PrivateKey: getEnvOrDefault("JWT_PRIVATE_KEY", ""),
		},
		HMAC: HMACConfig{
			Secret: getEnvOrDefault("HMAC_SECRET", ""),
		},
		Email: EmailConfig{
			SMTPEmail:    getEnvOrDefault("SMTP_EMAIL", ""),
			SMTPHost:     getEnvOrDefault("SMTP_HOST", ""),
			SMTPPort:     getEnvAsInt("SMTP_PORT", 587),
			SMTPUser:     getEnvOrDefault("SMTP_USER", ""),
			SMTPPass:     getEnvOrDefault("SMTP_PASS", ""),
			RefEmail:     getEnvOrDefault("REF_EMAIL", ""),
			RefEmailPass: getEnvOrDefault("REF_EMAIL_PASS", ""),
		},
		Security: SecurityConfig{
			RecaptchaSiteKey:  getEnvOrDefault("RECAPTCHA_SITE_KEY", ""),
			RecaptchaKey:      getEnvOrDefault("RECAPTCHA_KEY", ""),
			RecaptchaDisabled: getEnvAsBool("RECAPTCHA_DISABLED", false),
			Origin:            getEnvOrDefault("ORIGIN", ""),
		},
		App: AppConfig{
			WebDomain:      getEnvOrDefault("WEB_DOMAIN", "http://localhost:3000"),
			OrgName:        getEnvOrDefault("ORG_NAME", "Telar"),
			Name:           getEnvOrDefault("APP_NAME", "Telar"),
			OrgAvatar:      getEnvOrDefault("ORG_AVATAR", ""),
			QueryPrettyURL: getEnvAsBool("QUERY_PRETTY_URL", false),
		},
		External: ExternalConfig{
			GitHubClientID:    getEnvOrDefault("GITHUB_CLIENT_ID", ""),
			GitHubSecret:      getEnvOrDefault("GITHUB_SECRET", ""),
			GoogleClientID:    getEnvOrDefault("GOOGLE_CLIENT_ID", ""),
			GoogleSecret:      getEnvOrDefault("GOOGLE_SECRET", ""),
			PhoneSourceNumber: getEnvOrDefault("PHONE_SOURCE_NUMBER", ""),
			PhoneAuthToken:    getEnvOrDefault("PHONE_AUTH_TOKEN", ""),
			PhoneAuthId:       getEnvOrDefault("PHONE_AUTH_ID", ""),
		},
		Cache: CacheConfig{
			MaxMemory:       getEnvAsInt64("CACHE_MAX_MEMORY", 100*1024*1024), // 100MB default
			TTL:             getEnvAsDuration("CACHE_TTL", 1*time.Hour),       // 1 hour default
			Enabled:         getEnvAsBool("CACHE_ENABLED", true),
			Backend:         getEnvOrDefault("CACHE_BACKEND", "memory"),
			Prefix:          getEnvOrDefault("CACHE_PREFIX", "telar:"),
			CleanupInterval: getEnvAsDuration("CACHE_CLEANUP_INTERVAL", 5*time.Minute),
			Redis: RedisConfig{
				Host:         getEnvOrDefault("REDIS_HOST", "localhost"),
				Port:         getEnvAsInt("REDIS_PORT", 6379),
				Password:     getEnvOrDefault("REDIS_PASSWORD", ""),
				DB:           getEnvAsInt("REDIS_DB", 0),
				PoolSize:     getEnvAsInt("REDIS_POOL_SIZE", 10),
				MinIdle:      getEnvAsInt("REDIS_MIN_IDLE", 5),
				Address:      getEnvOrDefault("REDIS_ADDRESS", "localhost:6379"),
				Database:     getEnvAsInt("REDIS_DATABASE", 0),
				MinIdleConns: getEnvAsInt("REDIS_MIN_IDLE_CONNS", 5),
				MaxConnAge:   time.Duration(getEnvAsInt("REDIS_MAX_CONN_AGE", 300)) * time.Second,
				Cluster: ClusterConfig{
					Addrs:     []string{getEnvOrDefault("REDIS_CLUSTER_ADDRS", "localhost:6379")},
					Password:  getEnvOrDefault("REDIS_CLUSTER_PASSWORD", ""),
					Enabled:   getEnvAsBool("REDIS_CLUSTER_ENABLED", false),
					Addresses: []string{getEnvOrDefault("REDIS_CLUSTER_ADDRESSES", "localhost:6379")},
				},
			},
		},
		RateLimits: RateLimitsConfig{
			Signup: RateLimitConfig{
				Enabled:  getEnvAsBool("RATE_LIMIT_SIGNUP_ENABLED", true),
				Max:      getEnvAsInt("RATE_LIMIT_SIGNUP_MAX", 10),
				Duration: getEnvAsDuration("RATE_LIMIT_SIGNUP_DURATION", 1*time.Hour),
			},
			Login: RateLimitConfig{
				Enabled:  getEnvAsBool("RATE_LIMIT_LOGIN_ENABLED", true),
				Max:      getEnvAsInt("RATE_LIMIT_LOGIN_MAX", 5),
				Duration: getEnvAsDuration("RATE_LIMIT_LOGIN_DURATION", 15*time.Minute),
			},
			PasswordReset: RateLimitConfig{
				Enabled:  getEnvAsBool("RATE_LIMIT_PASSWORD_RESET_ENABLED", true),
				Max:      getEnvAsInt("RATE_LIMIT_PASSWORD_RESET_MAX", 3),
				Duration: getEnvAsDuration("RATE_LIMIT_PASSWORD_RESET_DURATION", 1*time.Hour),
			},
			Verification: RateLimitConfig{
				Enabled:  getEnvAsBool("RATE_LIMIT_VERIFICATION_ENABLED", true),
				Max:      getEnvAsInt("RATE_LIMIT_VERIFICATION_MAX", 10),
				Duration: getEnvAsDuration("RATE_LIMIT_VERIFICATION_DURATION", 15*time.Minute),
			},
		},
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// LoadFromMap loads configuration from an in-memory map.
// This is the primary helper for testing configuration logic in isolation
// without manipulating global environment variables.
func LoadFromMap(envMap map[string]string) (*Config, error) {
	// Helper to get a value from the map or a default.
	get := func(key, defaultValue string) string {
		if value, exists := envMap[key]; exists {
			return value
		}
		return defaultValue
	}

	// Helper to get an integer value from the map or a default.
	getInt := func(key string, defaultValue int) int {
		if value, exists := envMap[key]; exists {
			if intValue, err := strconv.Atoi(value); err == nil {
				return intValue
			}
		}
		return defaultValue
	}

	// Helper to get an int64 value from the map or a default.
	getInt64 := func(key string, defaultValue int64) int64 {
		if value, exists := envMap[key]; exists {
			if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
				return intValue
			}
		}
		return defaultValue
	}

	// Helper to get a boolean value from the map or a default.
	getBool := func(key string, defaultValue bool) bool {
		if value, exists := envMap[key]; exists {
			if boolValue, err := strconv.ParseBool(value); err == nil {
				return boolValue
			}
		}
		return defaultValue
	}

	// Helper to get a duration value from the map or a default.
	getDuration := func(key string, defaultValue time.Duration) time.Duration {
		if value, exists := envMap[key]; exists {
			if duration, err := time.ParseDuration(value); err == nil {
				return duration
			}
		}
		return defaultValue
	}

	// Validate required fields
	jwtPrivateKey := get("JWT_PRIVATE_KEY", "")
	if jwtPrivateKey == "" {
		return nil, fmt.Errorf("required configuration JWT_PRIVATE_KEY is not set")
	}

	jwtPublicKey := get("JWT_PUBLIC_KEY", "")
	if jwtPublicKey == "" {
		return nil, fmt.Errorf("required configuration JWT_PUBLIC_KEY is not set")
	}

	hmacSecret := get("HMAC_SECRET", "")
	if hmacSecret == "" {
		return nil, fmt.Errorf("required configuration HMAC_SECRET is not set")
	}

	config := &Config{
		Server: ServerConfig{
			Host:            get("HOST", "localhost"),
			Port:            getInt("SERVER_PORT", 8080),
			BaseRoute:       get("BASE_ROUTE", "/api"),
			Gateway:         get("GATEWAY", "http://localhost:8080"),
			InternalGateway: get("INTERNAL_GATEWAY", "http://localhost:8080"),
			WebDomain:       get("WEB_DOMAIN", "http://localhost:3000"),
			Debug:           getBool("DEBUG", false),
		},
		Database: DatabaseConfig{
			Type:                  get("DB_TYPE", "postgresql"),
			ForceNonTransactional: getBool("FORCE_NON_TRANSACTIONAL", false),
			Postgres: PostgreSQLConfig{
				Host:            get("POSTGRES_HOST", "localhost"),
				Port:            getInt("POSTGRES_PORT", 5432),
				Username:        get("POSTGRES_USERNAME", ""),
				Password:        get("POSTGRES_PASSWORD", ""),
				Database:        get("POSTGRES_DATABASE", "telar"),
				DSN:             get("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/telar_social_test?sslmode=disable&search_path=public"),
				SSLMode:         get("POSTGRES_SSL_MODE", "disable"),
				MaxOpenConns:    getInt("POSTGRES_MAX_OPEN_CONNS", 25),
				MaxIdleConns:    getInt("POSTGRES_MAX_IDLE_CONNS", 25),
				ConnMaxLifetime: time.Duration(getInt("POSTGRES_CONN_MAX_LIFETIME", 300)) * time.Second,
			},
		},
		JWT: JWTConfig{
			PublicKey:  jwtPublicKey,
			PrivateKey: jwtPrivateKey,
		},
		HMAC: HMACConfig{
			Secret: hmacSecret,
		},
		Email: EmailConfig{
			SMTPEmail:    get("SMTP_EMAIL", ""),
			SMTPHost:     get("SMTP_HOST", ""),
			SMTPPort:     getInt("SMTP_PORT", 587),
			SMTPUser:     get("SMTP_USER", ""),
			SMTPPass:     get("SMTP_PASS", ""),
			RefEmail:     get("REF_EMAIL", ""),
			RefEmailPass: get("REF_EMAIL_PASS", ""),
		},
		Security: SecurityConfig{
			RecaptchaSiteKey:  get("RECAPTCHA_SITE_KEY", ""),
			RecaptchaKey:      get("RECAPTCHA_KEY", ""),
			RecaptchaDisabled: getBool("RECAPTCHA_DISABLED", false),
			Origin:            get("ORIGIN", ""),
		},
		App: AppConfig{
			WebDomain:      get("WEB_DOMAIN", "http://localhost:3000"),
			OrgName:        get("ORG_NAME", "Telar"),
			Name:           get("APP_NAME", "Telar"),
			OrgAvatar:      get("ORG_AVATAR", ""),
			QueryPrettyURL: getBool("QUERY_PRETTY_URL", false),
		},
		External: ExternalConfig{
			GitHubClientID:    get("GITHUB_CLIENT_ID", ""),
			GitHubSecret:      get("GITHUB_SECRET", ""),
			GoogleClientID:    get("GOOGLE_CLIENT_ID", ""),
			GoogleSecret:      get("GOOGLE_SECRET", ""),
			PhoneSourceNumber: get("PHONE_SOURCE_NUMBER", ""),
			PhoneAuthToken:    get("PHONE_AUTH_TOKEN", ""),
			PhoneAuthId:       get("PHONE_AUTH_ID", ""),
		},
		Cache: CacheConfig{
			MaxMemory:       getInt64("CACHE_MAX_MEMORY", 100*1024*1024), // 100MB default
			TTL:             getDuration("CACHE_TTL", 1*time.Hour),       // 1 hour default
			Enabled:         getBool("CACHE_ENABLED", true),
			Backend:         get("CACHE_BACKEND", "memory"),
			Prefix:          get("CACHE_PREFIX", "telar:"),
			CleanupInterval: getDuration("CACHE_CLEANUP_INTERVAL", 5*time.Minute),
			Redis: RedisConfig{
				Host:         get("REDIS_HOST", "localhost"),
				Port:         getInt("REDIS_PORT", 6379),
				Password:     get("REDIS_PASSWORD", ""),
				DB:           getInt("REDIS_DB", 0),
				PoolSize:     getInt("REDIS_POOL_SIZE", 10),
				MinIdle:      getInt("REDIS_MIN_IDLE", 5),
				Address:      get("REDIS_ADDRESS", "localhost:6379"),
				Database:     getInt("REDIS_DATABASE", 0),
				MinIdleConns: getInt("REDIS_MIN_IDLE_CONNS", 5),
				MaxConnAge:   time.Duration(getInt("REDIS_MAX_CONN_AGE", 300)) * time.Second,
				Cluster: ClusterConfig{
					Addrs:     []string{get("REDIS_CLUSTER_ADDRS", "localhost:6379")},
					Password:  get("REDIS_CLUSTER_PASSWORD", ""),
					Enabled:   getBool("REDIS_CLUSTER_ENABLED", false),
					Addresses: []string{get("REDIS_CLUSTER_ADDRESSES", "localhost:6379")},
				},
			},
		},
		RateLimits: RateLimitsConfig{
			Signup: RateLimitConfig{
				Enabled:  getEnvAsBool("RATE_LIMIT_SIGNUP_ENABLED", true),
				Max:      getEnvAsInt("RATE_LIMIT_SIGNUP_MAX", 10),
				Duration: getEnvAsDuration("RATE_LIMIT_SIGNUP_DURATION", 1*time.Hour),
			},
			Login: RateLimitConfig{
				Enabled:  getEnvAsBool("RATE_LIMIT_LOGIN_ENABLED", true),
				Max:      getEnvAsInt("RATE_LIMIT_LOGIN_MAX", 5),
				Duration: getEnvAsDuration("RATE_LIMIT_LOGIN_DURATION", 15*time.Minute),
			},
			PasswordReset: RateLimitConfig{
				Enabled:  getEnvAsBool("RATE_LIMIT_PASSWORD_RESET_ENABLED", true),
				Max:      getEnvAsInt("RATE_LIMIT_PASSWORD_RESET_MAX", 3),
				Duration: getEnvAsDuration("RATE_LIMIT_PASSWORD_RESET_DURATION", 1*time.Hour),
			},
			Verification: RateLimitConfig{
				Enabled:  getEnvAsBool("RATE_LIMIT_VERIFICATION_ENABLED", true),
				Max:      getEnvAsInt("RATE_LIMIT_VERIFICATION_MAX", 10),
				Duration: getEnvAsDuration("RATE_LIMIT_VERIFICATION_DURATION", 15*time.Minute),
			},
		},
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// Validate validates the configuration for required fields
func (c *Config) Validate() error {
	var errors []string

	// Validate required JWT fields
	if strings.TrimSpace(c.JWT.PublicKey) == "" {
		errors = append(errors, "JWT_PUBLIC_KEY is required")
	}
	if strings.TrimSpace(c.JWT.PrivateKey) == "" {
		errors = append(errors, "JWT_PRIVATE_KEY is required")
	}

	// Validate required HMAC fields
	if strings.TrimSpace(c.HMAC.Secret) == "" {
		errors = append(errors, "HMAC_SECRET is required")
	}

	// Validate database type
	validDbTypes := []string{"postgresql"}
	if !contains(validDbTypes, c.Database.Type) {
		errors = append(errors, fmt.Sprintf("DB_TYPE must be one of: %s", strings.Join(validDbTypes, ", ")))
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	return getEnvAsInt(key, defaultValue)
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	return getEnvAsBool(key, defaultValue)
}

func getEnvDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	return getEnvAsDuration(key, defaultValue)
}
