// Package validation provides tests for enhanced input sanitization and validation
// Following the AUTH_SECURITY_REFACTORING_PLAN.md Phase 2.2 testing requirements
package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAndSanitizeEmail_ValidEmails(t *testing.T) {
	validEmails := []struct {
		input    string
		expected string
	}{
		{"test@example.com", "test@example.com"},
		{"  TEST@EXAMPLE.COM  ", "test@example.com"},
		{"user.name@domain.co.uk", "user.name@domain.co.uk"},
		{"user+tag@example.org", "user+tag@example.org"},
		{"firstname.lastname@subdomain.example.com", "firstname.lastname@subdomain.example.com"},
	}

	for _, test := range validEmails {
		t.Run(test.input, func(t *testing.T) {
			result, err := ValidateAndSanitizeEmail(test.input)
			require.NoError(t, err)
			assert.True(t, result.IsValid)
			assert.Equal(t, test.expected, result.SanitizedValue)
			assert.Empty(t, result.Errors)
		})
	}
}

func TestValidateAndSanitizeEmail_InvalidEmails(t *testing.T) {
	invalidEmails := []struct {
		input       string
		expectedErr string
	}{
		{"", "email is required"},
		{"invalid-email", "email must be a valid email address"},
		{"user@", "email must be a valid email address"},
		{"@domain.com", "email must be a valid email address"},
		{"user..name@domain.com", "email cannot contain consecutive dots"},
		{"user@domain", "email must have a valid domain with TLD"},
		{string(make([]byte, 300)), "email must be less than 254 characters"},
	}

	for _, test := range invalidEmails {
		t.Run(test.input, func(t *testing.T) {
			result, err := ValidateAndSanitizeEmail(test.input)
			require.NoError(t, err)
			assert.False(t, result.IsValid)
			if len(result.Errors) > 0 {
				assert.Contains(t, result.Errors[0], test.expectedErr)
			} else {
				t.Errorf("Expected validation error but got none for input: %s", test.input)
			}
		})
	}
}

func TestValidateAndSanitizeEmail_DisposableEmails(t *testing.T) {
	disposableEmails := []string{
		"test@10minutemail.com",
		"user@guerrillamail.com",
		"example@mailinator.com",
		"temp@tempmail.org",
		"spam@yopmail.com",
	}

	for _, email := range disposableEmails {
		t.Run(email, func(t *testing.T) {
			result, err := ValidateAndSanitizeEmail(email)
			require.NoError(t, err)
			assert.False(t, result.IsValid)
			assert.Contains(t, result.Errors[0], "disposable email addresses are not allowed")
		})
	}
}

func TestValidateAndSanitizeEmail_SuspiciousPatterns(t *testing.T) {
	suspiciousEmails := []string{
		"user@domain.com'; DROP TABLE users; --",
		"<script>alert('xss')</script>@domain.com",
		"user@domain.com UNION SELECT * FROM users",
		"user@domain.com || true",
	}

	for _, email := range suspiciousEmails {
		t.Run(email, func(t *testing.T) {
			result, err := ValidateAndSanitizeEmail(email)
			require.NoError(t, err)
			assert.False(t, result.IsValid)
			assert.Contains(t, result.Errors[0], "email contains invalid characters or patterns")
		})
	}
}

func TestValidateAndSanitizeFullName_ValidNames(t *testing.T) {
	validNames := []struct {
		input    string
		expected string
	}{
		{"John Doe", "John Doe"},
		{"  Mary Jane  ", "Mary Jane"},
		{"Jean-Pierre", "Jean-Pierre"},
		{"O'Connor", "O'Connor"},
		{"Maria José", "Maria José"},
		{"李小明", "李小明"},
		{"Anna-Maria Schmidt", "Anna-Maria Schmidt"},
	}

	for _, test := range validNames {
		t.Run(test.input, func(t *testing.T) {
			result, err := ValidateAndSanitizeFullName(test.input)
			require.NoError(t, err)
			assert.True(t, result.IsValid)
			assert.Equal(t, test.expected, result.SanitizedValue)
			assert.Empty(t, result.Errors)
		})
	}
}

func TestValidateAndSanitizeFullName_InvalidNames(t *testing.T) {
	invalidNames := []struct {
		input       string
		expectedErr string
	}{
		{"", "fullName is required"},
		{"A", "fullName must be at least 2 characters"},
		{string(make([]byte, 150)), "fullName must be less than 100 characters"},
		{"John123", "fullName contains invalid characters"},
		{"John@Doe", "fullName contains invalid characters"},
		{"<script>alert('xss')</script>", "fullName contains invalid characters or patterns"},
	}

	for _, test := range invalidNames {
		t.Run(test.input, func(t *testing.T) {
			result, err := ValidateAndSanitizeFullName(test.input)
			require.NoError(t, err)
			assert.False(t, result.IsValid)
			assert.Contains(t, result.Errors[0], test.expectedErr)
		})
	}
}

func TestValidateAndSanitizeFullName_ExcessiveSpaces(t *testing.T) {
	result, err := ValidateAndSanitizeFullName("John    Doe")
	require.NoError(t, err)
	assert.True(t, result.IsValid)
	assert.Equal(t, "John Doe", result.SanitizedValue)
	assert.Contains(t, result.Warnings[0], "excessive spaces detected and normalized")
}

func TestValidateAndSanitizeSocialName_ValidNames(t *testing.T) {
	validNames := []struct {
		input    string
		expected string
	}{
		{"johndoe", "johndoe"},
		{"  JOHN-DOE  ", "john-doe"},
		{"user123", "user123"},
		{"my-username", "my-username"},
		{"test-user-2024", "test-user-2024"},
	}

	for _, test := range validNames {
		t.Run(test.input, func(t *testing.T) {
			result, err := ValidateAndSanitizeSocialName(test.input)
			require.NoError(t, err)
			assert.True(t, result.IsValid)
			assert.Equal(t, test.expected, result.SanitizedValue)
			assert.Empty(t, result.Errors)
		})
	}
}

func TestValidateAndSanitizeSocialName_InvalidNames(t *testing.T) {
	invalidNames := []struct {
		input       string
		expectedErr string
	}{
		{"", "socialName is required"},
		{"ab", "socialName must be at least 3 characters"},
		{string(make([]byte, 60)), "socialName must be less than 50 characters"},
		{"-username", "socialName must contain only letters, numbers, and hyphens"},
		{"username-", "socialName must contain only letters, numbers, and hyphens"},
		{"user@name", "socialName must contain only letters, numbers, and hyphens"},
		{"user name", "socialName must contain only letters, numbers, and hyphens"},
	}

	for _, test := range invalidNames {
		t.Run(test.input, func(t *testing.T) {
			result, err := ValidateAndSanitizeSocialName(test.input)
			require.NoError(t, err)
			assert.False(t, result.IsValid)
			assert.Contains(t, result.Errors[0], test.expectedErr)
		})
	}
}

func TestValidateAndSanitizeSocialName_ReservedWords(t *testing.T) {
	reservedWords := []string{"admin", "root", "system", "api", "www", "mail"}

	for _, word := range reservedWords {
		t.Run(word, func(t *testing.T) {
			result, err := ValidateAndSanitizeSocialName(word)
			require.NoError(t, err)
			assert.False(t, result.IsValid)
			assert.Contains(t, result.Errors[0], "socialName cannot be a reserved word")
		})
	}
}

func TestValidateAndSanitizeURL_ValidURLs(t *testing.T) {
	validURLs := []struct {
		input    string
		expected string
	}{
		{"https://example.com", "https://example.com"},
		{"  https://example.com/path  ", "https://example.com/path"},
		{"https://cdn.example.com/image.jpg", "https://cdn.example.com/image.jpg"},
		{"https://api.example.com:9099/v1/endpoint", "https://api.example.com:9099/v1/endpoint"},
	}

	for _, test := range validURLs {
		t.Run(test.input, func(t *testing.T) {
			result, err := ValidateAndSanitizeURL(test.input)
			require.NoError(t, err)
			assert.True(t, result.IsValid)
			assert.Equal(t, test.expected, result.SanitizedValue)
			assert.Empty(t, result.Errors)
		})
	}
}

func TestValidateAndSanitizeURL_InvalidURLs(t *testing.T) {
	invalidURLs := []struct {
		input       string
		expectedErr string
	}{
		{"", "URL is required"},
		{"http://insecure.com", "URL must use HTTPS protocol"},
		{"ftp://example.com", "URL must use HTTPS protocol"},
		{string(make([]byte, 3000)), "URL must be less than 2048 characters"},
		{"https://example.com'; DROP TABLE users; --", "URL contains invalid characters or patterns"},
	}

	for _, test := range invalidURLs {
		t.Run(test.input, func(t *testing.T) {
			result, err := ValidateAndSanitizeURL(test.input)
			require.NoError(t, err)
			assert.False(t, result.IsValid)
			assert.Contains(t, result.Errors[0], test.expectedErr)
		})
	}
}

func TestContainsSuspiciousPatterns_SQLInjection(t *testing.T) {
	suspiciousInputs := []string{
		"'; DROP TABLE users; --",
		"UNION SELECT * FROM users",
		"<script>alert('xss')</script>",
		"javascript:alert('xss')",
		"1' OR '1'='1",
		"'; INSERT INTO users VALUES ('hacker'); --",
		"/* comment */ SELECT * FROM users",
	}

	for _, input := range suspiciousInputs {
		t.Run(input, func(t *testing.T) {
			result := containsSuspiciousPatterns(input)
			assert.True(t, result, "Should detect suspicious pattern in: %s", input)
		})
	}
}

func TestContainsSuspiciousPatterns_SafeInputs(t *testing.T) {
	safeInputs := []string{
		"john.doe@example.com",
		"Hello World!",
		"This is a normal string with 123 numbers",
		"User's full name with apostrophe",
		"https://example.com/path?param=value",
		"Normal text with (parentheses) and [brackets]",
	}

	for _, input := range safeInputs {
		t.Run(input, func(t *testing.T) {
			result := containsSuspiciousPatterns(input)
			assert.False(t, result, "Should not detect suspicious pattern in: %s", input)
		})
	}
}

func TestIsDisposableEmailDomain_KnownDisposable(t *testing.T) {
	disposableDomains := []string{
		"10minutemail.com",
		"guerrillamail.com",
		"mailinator.com",
		"tempmail.org",
		"yopmail.com",
	}

	for _, domain := range disposableDomains {
		t.Run(domain, func(t *testing.T) {
			result := isDisposableEmailDomain(domain)
			assert.True(t, result, "Should detect disposable domain: %s", domain)
		})
	}
}

func TestIsDisposableEmailDomain_LegitimateProviders(t *testing.T) {
	legitimateDomains := []string{
		"gmail.com",
		"yahoo.com",
		"outlook.com",
		"example.com",
		"company.org",
	}

	for _, domain := range legitimateDomains {
		t.Run(domain, func(t *testing.T) {
			result := isDisposableEmailDomain(domain)
			assert.False(t, result, "Should not detect legitimate domain as disposable: %s", domain)
		})
	}
}

// Integration tests for the enhanced validation functions

func TestValidateSignupTokenRequest_EnhancedSecurity(t *testing.T) {
	// This would require importing the signup models, but demonstrates the integration
	// In a real scenario, you would create a proper test with the actual signup.SignupTokenModel

	// Test case: suspicious email pattern
	// req := &signup.SignupTokenModel{
	//     User: signup.UserModel{
	//         Email: "test'; DROP TABLE users; --@example.com",
	//         Fullname: "Test User",
	//         Password: "validPassword123",
	//     },
	//     VerifyType: "email",
	//     Recaptcha: "valid-recaptcha-response",
	// }
	//
	// err := ValidateSignupTokenRequest(req)
	// assert.Error(t, err)
	// assert.Contains(t, err.Error(), "email validation failed")
}

func TestValidateLoginRequest_EnhancedSecurity(t *testing.T) {
	// This would require importing the login models
	// Similar integration test demonstrating the enhanced login validation
}

// Performance tests to ensure validation doesn't significantly impact performance
func BenchmarkValidateAndSanitizeEmail(b *testing.B) {
	email := "test.user@example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ValidateAndSanitizeEmail(email)
	}
}

func BenchmarkContainsSuspiciousPatterns(b *testing.B) {
	input := "This is a normal text string with no suspicious patterns"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = containsSuspiciousPatterns(input)
	}
}
