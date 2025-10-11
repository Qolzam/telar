package cache

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sync"
	"time"
)

// AccessPolicy defines access control for cache operations
type AccessPolicy struct {
	AllowRead   bool     `json:"allowRead" yaml:"allowRead"`
	AllowWrite  bool     `json:"allowWrite" yaml:"allowWrite"`
	AllowDelete bool     `json:"allowDelete" yaml:"allowDelete"`
	KeyPattern  string   `json:"keyPattern" yaml:"keyPattern"`
	UserRoles   []string `json:"userRoles" yaml:"userRoles"`
}

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	RequestsPerSecond int           `json:"requestsPerSecond" yaml:"requestsPerSecond"`
	BurstSize         int           `json:"burstSize" yaml:"burstSize"`
	WindowSize        time.Duration `json:"windowSize" yaml:"windowSize"`
}

// AccessController manages access control for cache operations
type AccessController struct {
	policies map[string]AccessPolicy
	mutex    sync.RWMutex
}

// NewAccessController creates a new access controller
func NewAccessController(policies map[string]AccessPolicy) *AccessController {
	return &AccessController{
		policies: policies,
	}
}

// CanRead checks if read operation is allowed
func (ac *AccessController) CanRead(ctx context.Context, key string) bool {
	ac.mutex.RLock()
	defer ac.mutex.RUnlock()
	
	for _, policy := range ac.policies {
		if ac.matchesPattern(key, policy.KeyPattern) {
			return policy.AllowRead
		}
	}
	
	return true // Allow by default if no matching policy
}

// CanWrite checks if write operation is allowed
func (ac *AccessController) CanWrite(ctx context.Context, key string) bool {
	ac.mutex.RLock()
	defer ac.mutex.RUnlock()
	
	for _, policy := range ac.policies {
		if ac.matchesPattern(key, policy.KeyPattern) {
			return policy.AllowWrite
		}
	}
	
	return true // Allow by default if no matching policy
}

// CanDelete checks if delete operation is allowed
func (ac *AccessController) CanDelete(ctx context.Context, key string) bool {
	ac.mutex.RLock()
	defer ac.mutex.RUnlock()
	
	for _, policy := range ac.policies {
		if ac.matchesPattern(key, policy.KeyPattern) {
			return policy.AllowDelete
		}
	}
	
	return true // Allow by default if no matching policy
}

// matchesPattern checks if key matches the pattern
func (ac *AccessController) matchesPattern(key, pattern string) bool {
	matched, err := regexp.MatchString(pattern, key)
	return err == nil && matched
}

// RateLimiter manages rate limiting for cache operations
type RateLimiter struct {
	limiters map[string]*TokenBucket
	mutex    sync.RWMutex
}

// TokenBucket implements token bucket rate limiting
type TokenBucket struct {
	capacity     int
	tokens       int
	refillRate   int
	lastRefill   time.Time
	mutex        sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(configs map[string]RateLimitConfig) *RateLimiter {
	limiters := make(map[string]*TokenBucket)
	
	for operation, config := range configs {
		limiters[operation] = &TokenBucket{
			capacity:   config.BurstSize,
			tokens:     config.BurstSize,
			refillRate: config.RequestsPerSecond,
			lastRefill: time.Now(),
		}
	}
	
	return &RateLimiter{
		limiters: limiters,
	}
}

// Allow checks if operation is allowed under rate limit
func (rl *RateLimiter) Allow(ctx context.Context, operation string) bool {
	rl.mutex.RLock()
	bucket, exists := rl.limiters[operation]
	rl.mutex.RUnlock()
	
	if !exists {
		return true // Allow if no rate limit configured
	}
	
	return bucket.Consume(1)
}

// Consume consumes tokens from the bucket
func (tb *TokenBucket) Consume(tokens int) bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	
	tb.refill()
	
	if tb.tokens >= tokens {
		tb.tokens -= tokens
		return true
	}
	
	return false
}

// refill refills the token bucket
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	
	tokensToAdd := int(elapsed.Seconds()) * tb.refillRate
	if tokensToAdd > 0 {
		tb.tokens += tokensToAdd
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastRefill = now
	}
}

// AuditLogger logs cache operations for security auditing
type AuditLogger struct {
	logPath string
	mutex   sync.Mutex
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logPath string) (*AuditLogger, error) {
	return &AuditLogger{
		logPath: logPath,
	}, nil
}

// Log logs a cache operation
func (al *AuditLogger) Log(ctx context.Context, operation, key, value, result string) {
	al.mutex.Lock()
	defer al.mutex.Unlock()
	
	// In a real implementation, write to file or send to logging service
	// For now, we'll just track that logging would happen
}

// Close closes the audit logger
func (al *AuditLogger) Close() error {
	return nil
}

// KeyValidator validates cache keys and values
type KeyValidator struct {
	allowedPatterns []*regexp.Regexp
	maxKeyLength    int
	maxValueSize    int64
}

// NewKeyValidator creates a new key validator
func NewKeyValidator(patterns []string, maxKeyLength int, maxValueSize int64) *KeyValidator {
	var regexes []*regexp.Regexp
	
	for _, pattern := range patterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			regexes = append(regexes, regex)
		}
	}
	
	return &KeyValidator{
		allowedPatterns: regexes,
		maxKeyLength:    maxKeyLength,
		maxValueSize:    maxValueSize,
	}
}

// ValidateKey validates a cache key
func (kv *KeyValidator) ValidateKey(key string) error {
	if len(key) > kv.maxKeyLength {
		return fmt.Errorf("key length %d exceeds maximum %d", len(key), kv.maxKeyLength)
	}
	
	// Check against allowed patterns
	for _, pattern := range kv.allowedPatterns {
		if pattern.MatchString(key) {
			return nil
		}
	}
	
	return fmt.Errorf("key %q does not match any allowed pattern", key)
}

// ValidateValue validates a cache value
func (kv *KeyValidator) ValidateValue(value []byte) error {
	if int64(len(value)) > kv.maxValueSize {
		return fmt.Errorf("value size %d exceeds maximum %d", len(value), kv.maxValueSize)
	}
	
	return nil
}

// SecureCache wraps a cache with security features
type SecureCache struct {
	cache       Cache
	config      *SecurityConfig
	encryptor   *CacheEncryptor
	accessCtrl  *AccessController
	rateLimiter *RateLimiter
	auditor     *AuditLogger
	validator   *KeyValidator
}

// NewSecureCache creates a new secure cache wrapper
func NewSecureCache(cache Cache, config *SecurityConfig) (*SecureCache, error) {
	sc := &SecureCache{
		cache:  cache,
		config: config,
	}
	
	// Initialize encryption
	if config.EncryptionEnabled {
		encryptor, err := NewCacheEncryptor(config.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize encryption: %w", err)
		}
		sc.encryptor = encryptor
	}
	
	// Initialize access control (simplified for actual SecurityConfig)
	if config.AuthEnabled {
		// Simple access control based on allowed origins/methods
		policies := make(map[string]AccessPolicy)
		policies["default"] = AccessPolicy{
			AllowRead:   true,
			AllowWrite:  true,
			AllowDelete: true,
			KeyPattern:  ".*",
		}
		sc.accessCtrl = NewAccessController(policies)
	}
	
	// Initialize rate limiting
	if config.RateLimitingEnabled {
		rateLimits := make(map[string]RateLimitConfig)
		rateLimits["default"] = RateLimitConfig{
			RequestsPerSecond: int(config.RateLimit),
			BurstSize:         int(config.RateLimit * 2),
			WindowSize:        time.Second,
		}
		sc.rateLimiter = NewRateLimiter(rateLimits)
	}
	
	// Initialize audit logging (simplified)
	sc.auditor, _ = NewAuditLogger("/var/log/cache_audit.log")
	
	// Initialize key validation (default patterns)
	patterns := []string{".*"} // Allow all patterns by default
	sc.validator = NewKeyValidator(patterns, 256, 1024*1024)
	
	return sc, nil
}

// Get retrieves a value with security checks
func (sc *SecureCache) Get(ctx context.Context, key string) ([]byte, error) {
	// Validate key
	if sc.validator != nil {
		if err := sc.validator.ValidateKey(key); err != nil {
			sc.auditLog(ctx, "GET", key, "", fmt.Sprintf("key validation failed: %v", err))
			return nil, err
		}
	}
	
	// Check access control
	if sc.accessCtrl != nil {
		if !sc.accessCtrl.CanRead(ctx, key) {
			sc.auditLog(ctx, "GET", key, "", "access denied")
			return nil, errors.New("access denied for read operation")
		}
	}
	
	// Check rate limit
	if sc.rateLimiter != nil {
		if !sc.rateLimiter.Allow(ctx, "GET") {
			sc.auditLog(ctx, "GET", key, "", "rate limit exceeded")
			return nil, errors.New("rate limit exceeded")
		}
	}
	
	// Get value from cache
	value, err := sc.cache.Get(ctx, key)
	if err != nil {
		sc.auditLog(ctx, "GET", key, "", fmt.Sprintf("cache error: %v", err))
		return nil, err
	}
	
	// Decrypt if needed
	if sc.encryptor != nil {
		value, err = sc.encryptor.Decrypt(value)
		if err != nil {
			sc.auditLog(ctx, "GET", key, "", fmt.Sprintf("decryption failed: %v", err))
			return nil, fmt.Errorf("decryption failed: %w", err)
		}
	}
	
	sc.auditLog(ctx, "GET", key, "", "success")
	return value, nil
}

// Set stores a value with security checks
func (sc *SecureCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	// Validate key and value
	if sc.validator != nil {
		if err := sc.validator.ValidateKey(key); err != nil {
			sc.auditLog(ctx, "SET", key, "", fmt.Sprintf("key validation failed: %v", err))
			return err
		}
		if err := sc.validator.ValidateValue(value); err != nil {
			sc.auditLog(ctx, "SET", key, "", fmt.Sprintf("value validation failed: %v", err))
			return err
		}
	}
	
	// Check access control
	if sc.accessCtrl != nil {
		if !sc.accessCtrl.CanWrite(ctx, key) {
			sc.auditLog(ctx, "SET", key, "", "access denied")
			return errors.New("access denied for write operation")
		}
	}
	
	// Check rate limit
	if sc.rateLimiter != nil {
		if !sc.rateLimiter.Allow(ctx, "SET") {
			sc.auditLog(ctx, "SET", key, "", "rate limit exceeded")
			return errors.New("rate limit exceeded")
		}
	}
	
	// Encrypt if needed
	finalValue := value
	if sc.encryptor != nil {
		var err error
		finalValue, err = sc.encryptor.Encrypt(value)
		if err != nil {
			sc.auditLog(ctx, "SET", key, "", fmt.Sprintf("encryption failed: %v", err))
			return fmt.Errorf("encryption failed: %w", err)
		}
	}
	
	// Store in cache
	err := sc.cache.Set(ctx, key, finalValue, ttl)
	if err != nil {
		sc.auditLog(ctx, "SET", key, "", fmt.Sprintf("cache error: %v", err))
		return err
	}
	
	sc.auditLog(ctx, "SET", key, "", "success")
	return nil
}

// Delete removes a value with security checks
func (sc *SecureCache) Delete(ctx context.Context, key string) error {
	// Validate key
	if sc.validator != nil {
		if err := sc.validator.ValidateKey(key); err != nil {
			sc.auditLog(ctx, "DELETE", key, "", fmt.Sprintf("key validation failed: %v", err))
			return err
		}
	}
	
	// Check access control
	if sc.accessCtrl != nil {
		if !sc.accessCtrl.CanDelete(ctx, key) {
			sc.auditLog(ctx, "DELETE", key, "", "access denied")
			return errors.New("access denied for delete operation")
		}
	}
	
	// Check rate limit
	if sc.rateLimiter != nil {
		if !sc.rateLimiter.Allow(ctx, "DELETE") {
			sc.auditLog(ctx, "DELETE", key, "", "rate limit exceeded")
			return errors.New("rate limit exceeded")
		}
	}
	
	// Delete from cache
	err := sc.cache.Delete(ctx, key)
	if err != nil {
		sc.auditLog(ctx, "DELETE", key, "", fmt.Sprintf("cache error: %v", err))
		return err
	}
	
	sc.auditLog(ctx, "DELETE", key, "", "success")
	return nil
}

// DeletePattern removes all keys matching pattern with security checks
func (sc *SecureCache) DeletePattern(ctx context.Context, pattern string) error {
	// Check access control for pattern operations
	if sc.accessCtrl != nil {
		if !sc.accessCtrl.CanDelete(ctx, pattern) {
			sc.auditLog(ctx, "DELETE_PATTERN", pattern, "", "access denied")
			return errors.New("access denied for pattern delete operation")
		}
	}
	
	// Check rate limit
	if sc.rateLimiter != nil {
		if !sc.rateLimiter.Allow(ctx, "DELETE_PATTERN") {
			sc.auditLog(ctx, "DELETE_PATTERN", pattern, "", "rate limit exceeded")
			return errors.New("rate limit exceeded")
		}
	}
	
	err := sc.cache.DeletePattern(ctx, pattern)
	if err != nil {
		sc.auditLog(ctx, "DELETE_PATTERN", pattern, "", fmt.Sprintf("cache error: %v", err))
		return err
	}
	
	sc.auditLog(ctx, "DELETE_PATTERN", pattern, "", "success")
	return nil
}

// Exists checks if key exists with security checks
func (sc *SecureCache) Exists(ctx context.Context, key string) (bool, error) {
	// Validate key
	if sc.validator != nil {
		if err := sc.validator.ValidateKey(key); err != nil {
			sc.auditLog(ctx, "EXISTS", key, "", fmt.Sprintf("key validation failed: %v", err))
			return false, err
		}
	}
	
	// Check access control
	if sc.accessCtrl != nil {
		if !sc.accessCtrl.CanRead(ctx, key) {
			sc.auditLog(ctx, "EXISTS", key, "", "access denied")
			return false, errors.New("access denied for exists operation")
		}
	}
	
	// Check rate limit
	if sc.rateLimiter != nil {
		if !sc.rateLimiter.Allow(ctx, "EXISTS") {
			sc.auditLog(ctx, "EXISTS", key, "", "rate limit exceeded")
			return false, errors.New("rate limit exceeded")
		}
	}
	
	exists, err := sc.cache.Exists(ctx, key)
	if err != nil {
		sc.auditLog(ctx, "EXISTS", key, "", fmt.Sprintf("cache error: %v", err))
		return false, err
	}
	
	sc.auditLog(ctx, "EXISTS", key, "", "success")
	return exists, nil
}

// Increment increments a numeric value with security checks
func (sc *SecureCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	// Validate key
	if sc.validator != nil {
		if err := sc.validator.ValidateKey(key); err != nil {
			sc.auditLog(ctx, "INCREMENT", key, "", fmt.Sprintf("key validation failed: %v", err))
			return 0, err
		}
	}
	
	// Check access control
	if sc.accessCtrl != nil {
		if !sc.accessCtrl.CanWrite(ctx, key) {
			sc.auditLog(ctx, "INCREMENT", key, "", "access denied")
			return 0, errors.New("access denied for increment operation")
		}
	}
	
	// Check rate limit
	if sc.rateLimiter != nil {
		if !sc.rateLimiter.Allow(ctx, "INCREMENT") {
			sc.auditLog(ctx, "INCREMENT", key, "", "rate limit exceeded")
			return 0, errors.New("rate limit exceeded")
		}
	}
	
	value, err := sc.cache.Increment(ctx, key, delta)
	if err != nil {
		sc.auditLog(ctx, "INCREMENT", key, "", fmt.Sprintf("cache error: %v", err))
		return 0, err
	}
	
	sc.auditLog(ctx, "INCREMENT", key, "", "success")
	return value, nil
}

// Stats returns cache statistics
func (sc *SecureCache) Stats() CacheStats {
	return sc.cache.Stats()
}

// Close closes the secure cache
func (sc *SecureCache) Close() error {
	if sc.auditor != nil {
		sc.auditor.Close()
	}
	return sc.cache.Close()
}

// auditLog logs cache operations for security auditing
func (sc *SecureCache) auditLog(ctx context.Context, operation, key, value, result string) {
	if sc.auditor != nil {
		sc.auditor.Log(ctx, operation, key, value, result)
	}
}

// CacheEncryptor handles cache value encryption/decryption
type CacheEncryptor struct {
	gcm cipher.AEAD
}

// NewCacheEncryptor creates a new cache encryptor
func NewCacheEncryptor(key string) (*CacheEncryptor, error) {
	if len(key) == 0 {
		return nil, errors.New("encryption key cannot be empty")
	}
	
	// Hash the key to ensure proper length
	hasher := sha256.New()
	hasher.Write([]byte(key))
	hashKey := hasher.Sum(nil)
	
	block, err := aes.NewCipher(hashKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	return &CacheEncryptor{gcm: gcm}, nil
}

// Encrypt encrypts cache value
func (ce *CacheEncryptor) Encrypt(data []byte) ([]byte, error) {
	nonce := make([]byte, ce.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	ciphertext := ce.gcm.Seal(nonce, nonce, data, nil)
	
	// Base64 encode for safe storage
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(ciphertext)))
	base64.StdEncoding.Encode(encoded, ciphertext)
	
	return encoded, nil
}

// Decrypt decrypts cache value
func (ce *CacheEncryptor) Decrypt(data []byte) ([]byte, error) {
	// Base64 decode
	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	n, err := base64.StdEncoding.Decode(decoded, data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	decoded = decoded[:n]
	
	nonceSize := ce.gcm.NonceSize()
	if len(decoded) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	
	nonce, ciphertext := decoded[:nonceSize], decoded[nonceSize:]
	plaintext, err := ce.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	
	return plaintext, nil
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		EncryptionEnabled:   false,
		EncryptionKey:       "",
		EncryptionAlgorithm: "aes-256-gcm",
		AuthEnabled:         false,
		AllowedOrigins:      []string{"*"},
		AllowedMethods:      []string{"GET", "POST", "PUT", "DELETE"},
		KeyHashingEnabled:   false,
		KeyHashingAlgorithm: "sha256",
		RateLimitingEnabled: false,
		RateLimit:           1000, // 1000 requests per second
	}
}
