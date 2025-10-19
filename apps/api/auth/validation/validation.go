package validation

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"

	"github.com/qolzam/telar/apps/api/auth/login"
	"github.com/qolzam/telar/apps/api/auth/password"
	"github.com/qolzam/telar/apps/api/auth/signup"
	"github.com/qolzam/telar/apps/api/auth/verification"
	"github.com/qolzam/telar/apps/api/internal/pkg/log"
	profileModels "github.com/qolzam/telar/apps/api/profile/models"
)

// Enhanced security validation configuration
var (
	// Common disposable email domains to block
	disposableEmailDomains = map[string]bool{
		"10minutemail.com":       true,
		"guerrillamail.com":      true,
		"mailinator.com":         true,
		"tempmail.org":           true,
		"temp-mail.org":          true,
		"throwaway.email":        true,
		"dispostable.com":        true,
		"yopmail.com":            true,
		"maildrop.cc":            true,
		"sharklasers.com":        true,
		"guerrillamailblock.com": true,
		"pokemail.net":           true,
		"spam4.me":               true,
		"bccto.me":               true,
		"mintemail.com":          true,
		"tempail.com":            true,
		"33mail.com":             true,
		"emailondeck.com":        true,
		"mailnesia.com":          true,
		"trashmail.com":          true,
	}

	// Email validation regex pattern (RFC 5322 compliant)
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

	// Username/social name validation patterns
	socialNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,48}[a-zA-Z0-9]$`)

	// Common SQL injection patterns for detection
	sqlInjectionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute)`),
		regexp.MustCompile(`(?i)(script|javascript|vbscript|onload|onerror|onclick)`),
		regexp.MustCompile(`(?i)(<|>|&lt;|&gt;|%3c|%3e)`),
		regexp.MustCompile(`(?i)(--|\||;|\/\*|\*\/)`),
		regexp.MustCompile(`(?i)('.*or.*'.*=.*'|".*or.*".*=.*")`), // OR patterns like '1'='1'
	}

	// Common weak passwords to reject
	commonWeakPasswords = map[string]bool{
		"password":    true,
		"123456":      true,
		"12345678":    true,
		"qwerty":      true,
		"abc123":      true,
		"password123": true,
		"admin":       true,
		"letmein":     true,
		"welcome":     true,
		"monkey":      true,
		"1234567890":  true,
		"password1":   true,
		"qwerty123":   true,
		"123123":      true,
		"000000":      true,
		"iloveyou":    true,
		"dragon":      true,
		"sunshine":    true,
		"princess":    true,
		"azerty":      true,
		"trustno1":    true,
		"123456789":   true,
	}

	// Character classes for entropy calculation
	lowercaseChars = "abcdefghijklmnopqrstuvwxyz"
	uppercaseChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitChars     = "0123456789"
	symbolChars    = "!@#$%^&*()_+-=[]{}|;:,.<>?"
)

// SecurityValidationResult contains the result of enhanced security validation
type SecurityValidationResult struct {
	IsValid        bool
	Errors         []string
	Warnings       []string
	SanitizedValue string
}

// PasswordPolicy defines comprehensive password requirements
type PasswordPolicy struct {
	MinLength        int     `json:"minLength"`        // Minimum password length
	MaxLength        int     `json:"maxLength"`        // Maximum password length
	RequireUppercase bool    `json:"requireUppercase"` // Must contain uppercase letters
	RequireLowercase bool    `json:"requireLowercase"` // Must contain lowercase letters
	RequireNumbers   bool    `json:"requireNumbers"`   // Must contain numbers
	RequireSymbols   bool    `json:"requireSymbols"`   // Must contain symbols
	MinEntropy       float64 `json:"minEntropy"`       // Minimum Shannon entropy
	MaxRepeating     int     `json:"maxRepeating"`     // Max consecutive repeating characters
	PreventCommon    bool    `json:"preventCommon"`    // Block common weak passwords
	PreventPersonal  bool    `json:"preventPersonal"`  // Block passwords containing personal info
}

// PasswordValidationResult contains detailed password validation results
type PasswordValidationResult struct {
	IsValid        bool     `json:"isValid"`
	Score          int      `json:"score"`          // 0-4 strength score
	Entropy        float64  `json:"entropy"`        // Shannon entropy
	Errors         []string `json:"errors"`         // Validation errors
	Warnings       []string `json:"warnings"`       // Security warnings
	Suggestions    []string `json:"suggestions"`    // Improvement suggestions
	EstimatedCrack string   `json:"estimatedCrack"` // Time to crack estimate
}

// ValidateAndSanitizeEmail performs comprehensive email validation and sanitization
func ValidateAndSanitizeEmail(email string) (*SecurityValidationResult, error) {
	result := &SecurityValidationResult{
		IsValid:  true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// 1. Basic sanitization - trim whitespace and convert to lowercase
	sanitizedEmail := strings.TrimSpace(strings.ToLower(email))
	result.SanitizedValue = sanitizedEmail

	// 2. Check for empty email
	if sanitizedEmail == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, "email is required")
		return result, nil
	}

	// 3. Check length constraints
	if len(sanitizedEmail) > 254 {
		result.IsValid = false
		result.Errors = append(result.Errors, "email must be less than 254 characters")
		return result, nil
	}

	// 4. Check for suspicious patterns early to give specific error messages
	if containsSuspiciousPatterns(sanitizedEmail) {
		result.IsValid = false
		result.Errors = append(result.Errors, "email contains invalid characters or patterns")
		log.Warn("[Security] Suspicious email pattern detected: %s", sanitizedEmail)
		return result, nil
	}

	// 5. Check for consecutive dots
	if strings.Contains(sanitizedEmail, "..") {
		result.IsValid = false
		result.Errors = append(result.Errors, "email cannot contain consecutive dots")
		return result, nil
	}

	// 6. Validate email format using Go's standard library
	if _, err := mail.ParseAddress(sanitizedEmail); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, "email must be a valid email address")
		return result, nil
	}

	// 7. Additional regex validation for stricter format checking
	if !emailRegex.MatchString(sanitizedEmail) {
		result.IsValid = false
		result.Errors = append(result.Errors, "email format is invalid")
		return result, nil
	}

	// 8. Extract domain for further validation
	parts := strings.Split(sanitizedEmail, "@")
	if len(parts) != 2 {
		result.IsValid = false
		result.Errors = append(result.Errors, "email must contain exactly one @ symbol")
		return result, nil
	}

	domain := parts[1]

	// 8a. Ensure domain has at least one dot (TLD requirement)
	if !strings.Contains(domain, ".") {
		result.IsValid = false
		result.Errors = append(result.Errors, "email must have a valid domain with TLD")
		return result, nil
	}

	// 9. Check against disposable email domains
	if isDisposableEmailDomain(domain) {
		result.IsValid = false
		result.Errors = append(result.Errors, "disposable email addresses are not allowed")
		log.Warn("[Security] Disposable email attempt: %s", sanitizedEmail)
		return result, nil
	}

	// 10. Domain length validation
	if len(domain) > 253 {
		result.IsValid = false
		result.Errors = append(result.Errors, "email domain is too long")
		return result, nil
	}

	return result, nil
}

// ValidateAndSanitizeFullName performs comprehensive full name validation and sanitization
func ValidateAndSanitizeFullName(fullName string) (*SecurityValidationResult, error) {
	result := &SecurityValidationResult{
		IsValid:  true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// 1. Basic sanitization - trim whitespace
	sanitized := strings.TrimSpace(fullName)
	result.SanitizedValue = sanitized

	// 2. Check for empty name
	if sanitized == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, "fullName is required")
		return result, nil
	}

	// 3. Length validation
	if len(sanitized) > 100 {
		result.IsValid = false
		result.Errors = append(result.Errors, "fullName must be less than 100 characters")
		return result, nil
	}

	if len(sanitized) < 2 {
		result.IsValid = false
		result.Errors = append(result.Errors, "fullName must be at least 2 characters")
		return result, nil
	}

	// 4. Check for suspicious patterns
	if containsSuspiciousPatterns(sanitized) {
		result.IsValid = false
		result.Errors = append(result.Errors, "fullName contains invalid characters or patterns")
		log.Warn("[Security] Suspicious full name pattern detected: %s", sanitized)
		return result, nil
	}

	// 5. Check for valid name characters (letters, spaces, common punctuation)
	for _, char := range sanitized {
		if !unicode.IsLetter(char) && !unicode.IsSpace(char) &&
			char != '.' && char != '-' && char != '\'' && char != ',' {
			result.IsValid = false
			result.Errors = append(result.Errors, "fullName contains invalid characters")
			return result, nil
		}
	}

	// 6. Check for excessive consecutive spaces
	if strings.Contains(sanitized, "  ") {
		result.Warnings = append(result.Warnings, "excessive spaces detected and normalized")
		// Normalize multiple spaces to single space
		spaceRegex := regexp.MustCompile(`\s+`)
		result.SanitizedValue = spaceRegex.ReplaceAllString(sanitized, " ")
	}

	return result, nil
}

// ValidateAndSanitizeSocialName performs social name validation and sanitization
func ValidateAndSanitizeSocialName(socialName string) (*SecurityValidationResult, error) {
	result := &SecurityValidationResult{
		IsValid:  true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// 1. Basic sanitization - trim whitespace and convert to lowercase
	sanitized := strings.TrimSpace(strings.ToLower(socialName))
	result.SanitizedValue = sanitized

	// 2. Check for empty name
	if sanitized == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, "socialName is required")
		return result, nil
	}

	// 3. Length validation
	if len(sanitized) > 50 {
		result.IsValid = false
		result.Errors = append(result.Errors, "socialName must be less than 50 characters")
		return result, nil
	}

	if len(sanitized) < 3 {
		result.IsValid = false
		result.Errors = append(result.Errors, "socialName must be at least 3 characters")
		return result, nil
	}

	// 4. Validate format using regex (alphanumeric and hyphens, not starting/ending with hyphen)
	if !socialNameRegex.MatchString(sanitized) {
		result.IsValid = false
		result.Errors = append(result.Errors, "socialName must contain only letters, numbers, and hyphens (not at start/end)")
		return result, nil
	}

	// 5. Check for suspicious patterns
	if containsSuspiciousPatterns(sanitized) {
		result.IsValid = false
		result.Errors = append(result.Errors, "socialName contains invalid patterns")
		log.Warn("[Security] Suspicious social name pattern detected: %s", sanitized)
		return result, nil
	}

	// 6. Check for reserved words
	reservedWords := []string{"admin", "root", "system", "api", "www", "mail", "ftp", "null", "undefined"}
	for _, word := range reservedWords {
		if sanitized == word {
			result.IsValid = false
			result.Errors = append(result.Errors, "socialName cannot be a reserved word")
			return result, nil
		}
	}

	return result, nil
}

// isDisposableEmailDomain checks if the domain is a known disposable email provider
func isDisposableEmailDomain(domain string) bool {
	return disposableEmailDomains[strings.ToLower(domain)]
}

// containsSuspiciousPatterns checks for SQL injection and XSS patterns
func containsSuspiciousPatterns(input string) bool {
	for _, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// ValidateAndSanitizeURL performs URL validation and sanitization
func ValidateAndSanitizeURL(url string) (*SecurityValidationResult, error) {
	result := &SecurityValidationResult{
		IsValid:  true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// 1. Basic sanitization - trim whitespace
	sanitized := strings.TrimSpace(url)
	result.SanitizedValue = sanitized

	// 2. Check for empty URL
	if sanitized == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, "URL is required")
		return result, nil
	}

	// 3. Length validation
	if len(sanitized) > 2048 {
		result.IsValid = false
		result.Errors = append(result.Errors, "URL must be less than 2048 characters")
		return result, nil
	}

	// 4. Validate URL scheme (only allow HTTPS for security)
	if !strings.HasPrefix(sanitized, "https://") {
		result.IsValid = false
		result.Errors = append(result.Errors, "URL must use HTTPS protocol")
		return result, nil
	}

	// 5. Check for suspicious patterns
	if containsSuspiciousPatterns(sanitized) {
		result.IsValid = false
		result.Errors = append(result.Errors, "URL contains invalid characters or patterns")
		log.Warn("[Security] Suspicious URL pattern detected: %s", sanitized)
		return result, nil
	}

	return result, nil
}

// ValidateSignupTokenRequest validates the signup token request with enhanced security
func ValidateSignupTokenRequest(req *signup.SignupTokenModel) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}

	// Enhanced full name validation
	fullNameResult, err := ValidateAndSanitizeFullName(req.User.Fullname)
	if err != nil {
		return fmt.Errorf("fullName validation error: %w", err)
	}
	if !fullNameResult.IsValid {
		return fmt.Errorf("fullName validation failed: %s", strings.Join(fullNameResult.Errors, ", "))
	}
	// Apply sanitized value back to the request
	req.User.Fullname = fullNameResult.SanitizedValue

	// Enhanced email validation
	emailResult, err := ValidateAndSanitizeEmail(req.User.Email)
	if err != nil {
		return fmt.Errorf("email validation error: %w", err)
	}
	if !emailResult.IsValid {
		return fmt.Errorf("email validation failed: %s", strings.Join(emailResult.Errors, ", "))
	}
	// Apply sanitized value back to the request
	req.User.Email = emailResult.SanitizedValue

	// Enhanced password validation (basic checks here, advanced in password policy)
	if req.User.Password == "" {
		return fmt.Errorf("password is required")
	}
	if len(req.User.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if containsSuspiciousPatterns(req.User.Password) {
		log.Warn("[Security] Suspicious password pattern detected for user: %s", req.User.Email)
		return fmt.Errorf("password contains invalid characters or patterns")
	}

	// Validate verify type with enhanced checks
	req.VerifyType = strings.TrimSpace(strings.ToLower(req.VerifyType))
	if req.VerifyType == "" {
		return fmt.Errorf("verifyType is required")
	}
	validVerifyTypes := []string{"email", "phone"}
	isValidVerifyType := false
	for _, vt := range validVerifyTypes {
		if req.VerifyType == vt {
			isValidVerifyType = true
			break
		}
	}
	if !isValidVerifyType {
		return fmt.Errorf("verifyType must be one of: %v", validVerifyTypes)
	}

	// Enhanced recaptcha validation
	req.Recaptcha = strings.TrimSpace(req.Recaptcha)
	if req.Recaptcha == "" {
		return fmt.Errorf("g-recaptcha-response is required")
	}
	if containsSuspiciousPatterns(req.Recaptcha) {
		log.Warn("[Security] Suspicious recaptcha pattern detected for user: %s", req.User.Email)
		return fmt.Errorf("recaptcha response contains invalid characters")
	}

	// Enhanced response type validation
	if req.ResponseType != "" {
		req.ResponseType = strings.TrimSpace(strings.ToLower(req.ResponseType))
		if req.ResponseType != "spa" && req.ResponseType != "ssr" {
			return fmt.Errorf("responseType must be 'spa' or 'ssr'")
		}
	}

	return nil
}

// ValidateLoginRequest validates the login request with enhanced security
func ValidateLoginRequest(req *login.LoginRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}

	// Enhanced username validation (could be email or username)
	req.Username = strings.TrimSpace(strings.ToLower(req.Username))
	if req.Username == "" {
		return fmt.Errorf("username is required")
	}

	// Check for suspicious patterns in username
	if containsSuspiciousPatterns(req.Username) {
		log.Warn("[Security] Suspicious username pattern detected: %s", req.Username)
		return fmt.Errorf("username contains invalid characters or patterns")
	}

	// Enhanced password validation
	if req.Password == "" {
		return fmt.Errorf("password is required")
	}

	// Check for suspicious patterns in password
	if containsSuspiciousPatterns(req.Password) {
		log.Warn("[Security] Suspicious password pattern detected for user: %s", req.Username)
		return fmt.Errorf("password contains invalid characters or patterns")
	}

	return nil
}

// ValidateChangePasswordRequest validates the change password request
func ValidateChangePasswordRequest(req *password.ChangePasswordRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}

	if req.OldPassword == "" {
		return fmt.Errorf("oldPassword is required")
	}

	if req.NewPassword == "" {
		return fmt.Errorf("newPassword is required")
	}
	if len(req.NewPassword) < 8 {
		return fmt.Errorf("newPassword must be at least 8 characters")
	}

	return nil
}

// ValidateForgetPasswordRequest validates the forget password request with enhanced security
func ValidateForgetPasswordRequest(req *password.ForgetPasswordRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}

	// Enhanced email validation
	emailResult, err := ValidateAndSanitizeEmail(req.Email)
	if err != nil {
		return fmt.Errorf("email validation error: %w", err)
	}
	if !emailResult.IsValid {
		return fmt.Errorf("email validation failed: %s", strings.Join(emailResult.Errors, ", "))
	}
	// Apply sanitized value back to the request
	req.Email = emailResult.SanitizedValue

	// Enhanced recaptcha validation
	req.Recaptcha = strings.TrimSpace(req.Recaptcha)
	if req.Recaptcha == "" {
		return fmt.Errorf("g-recaptcha-response is required")
	}
	if containsSuspiciousPatterns(req.Recaptcha) {
		log.Warn("[Security] Suspicious recaptcha pattern detected for password reset: %s", req.Email)
		return fmt.Errorf("recaptcha response contains invalid characters")
	}

	return nil
}

// ValidateResetPasswordFormRequest validates the reset password form request
func ValidateResetPasswordFormRequest(req *password.ResetPasswordFormRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}

	if req.NewPassword == "" {
		return fmt.Errorf("newPassword is required")
	}
	if len(req.NewPassword) < 8 {
		return fmt.Errorf("newPassword must be at least 8 characters")
	}

	return nil
}

// ValidateProfileUpdateRequest validates the profile update request with enhanced security
func ValidateProfileUpdateRequest(req *profileModels.UpdateProfileRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}

	// Enhanced fullName validation if provided
	if req.FullName != nil && *req.FullName != "" {
		fullNameResult, err := ValidateAndSanitizeFullName(*req.FullName)
		if err != nil {
			return fmt.Errorf("fullName validation error: %w", err)
		}
		if !fullNameResult.IsValid {
			return fmt.Errorf("fullName validation failed: %s", strings.Join(fullNameResult.Errors, ", "))
		}
		// Apply sanitized value back to the request
		*req.FullName = fullNameResult.SanitizedValue
	}

	// Enhanced socialName validation if provided
	if req.SocialName != nil && *req.SocialName != "" {
		socialNameResult, err := ValidateAndSanitizeSocialName(*req.SocialName)
		if err != nil {
			return fmt.Errorf("socialName validation error: %w", err)
		}
		if !socialNameResult.IsValid {
			return fmt.Errorf("socialName validation failed: %s", strings.Join(socialNameResult.Errors, ", "))
		}
		// Apply sanitized value back to the request
		*req.SocialName = socialNameResult.SanitizedValue
	}

	// Enhanced tagLine validation if provided
	if req.TagLine != nil && *req.TagLine != "" {
		tagLine := strings.TrimSpace(*req.TagLine)
		if len(tagLine) > 200 {
			return fmt.Errorf("tagLine must be less than 200 characters")
		}
		if containsSuspiciousPatterns(tagLine) {
			log.Warn("[Security] Suspicious tagLine pattern detected")
			return fmt.Errorf("tagLine contains invalid characters or patterns")
		}
		*req.TagLine = tagLine
	}

	// Enhanced avatar URL validation if provided
	if req.Avatar != nil && *req.Avatar != "" {
		avatarResult, err := ValidateAndSanitizeURL(*req.Avatar)
		if err != nil {
			return fmt.Errorf("avatar URL validation error: %w", err)
		}
		if !avatarResult.IsValid {
			return fmt.Errorf("avatar URL validation failed: %s", strings.Join(avatarResult.Errors, ", "))
		}
		*req.Avatar = avatarResult.SanitizedValue
	}

	// Enhanced banner URL validation if provided
	if req.Banner != nil && *req.Banner != "" {
		bannerResult, err := ValidateAndSanitizeURL(*req.Banner)
		if err != nil {
			return fmt.Errorf("banner URL validation error: %w", err)
		}
		if !bannerResult.IsValid {
			return fmt.Errorf("banner URL validation failed: %s", strings.Join(bannerResult.Errors, ", "))
		}
		*req.Banner = bannerResult.SanitizedValue
	}

	return nil
}

// ValidateVerifySignupRequest validates the verify signup request with enhanced security
func ValidateVerifySignupRequest(req *verification.VerifySignupRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}

	// Enhanced token validation
	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" {
		return fmt.Errorf("token is required")
	}
	if containsSuspiciousPatterns(req.Token) {
		log.Warn("[Security] Suspicious token pattern detected in verification")
		return fmt.Errorf("token contains invalid characters or patterns")
	}

	// Enhanced code validation
	req.Code = strings.TrimSpace(req.Code)
	if req.Code == "" {
		return fmt.Errorf("code is required")
	}
	if len(req.Code) != 6 {
		return fmt.Errorf("code must be exactly 6 characters")
	}
	// Validate that code contains only digits
	for _, char := range req.Code {
		if !unicode.IsDigit(char) {
			return fmt.Errorf("code must contain only digits")
		}
	}

	// Enhanced response type validation
	if req.ResponseType != "" {
		req.ResponseType = strings.TrimSpace(strings.ToLower(req.ResponseType))
		if req.ResponseType != "spa" && req.ResponseType != "ssr" {
			return fmt.Errorf("responseType must be 'spa' or 'ssr'")
		}
	}

	return nil
}
