package validation

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/qolzam/telar/apps/api/profile/models"
)

// Validation constants for social names
const (
	socialNameMinLength = 3
	socialNameMaxLength = 50
)

// Regular expressions for validation
var (
	// socialNamePattern allows alphanumeric characters, underscores, hyphens, and periods
	// Must start and end with alphanumeric, no consecutive special characters
	socialNamePattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9_.-]*[a-zA-Z0-9])?$`)
	
	// consecutiveSpecialChars prevents multiple consecutive special characters
	consecutiveSpecialChars = regexp.MustCompile(`[_.-]{2,}`)
)

func ValidateCreateProfileRequest(req *models.CreateProfileRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}

	if req.ObjectId.String() == "" {
		return fmt.Errorf("objectId is required")
	}

	if req.FullName != nil {
		if strings.TrimSpace(*req.FullName) == "" {
			return fmt.Errorf("fullName cannot be empty or whitespace only")
		}
		if len(*req.FullName) > 100 {
			return fmt.Errorf("fullName cannot exceed 100 characters")
		}
	}

	if req.SocialName != nil && *req.SocialName != "" {
		if err := ValidateSocialName(*req.SocialName); err != nil {
			return fmt.Errorf("socialName validation failed: %w", err)
		}
	}

	if req.Email != nil {
		if *req.Email != "" {
			if !isValidEmail(*req.Email) {
				return fmt.Errorf("invalid email format")
			}
			if len(*req.Email) > 255 {
				return fmt.Errorf("email cannot exceed 255 characters")
			}
		}
	}

	if req.Avatar != nil {
		if *req.Avatar != "" {
			if !isValidURL(*req.Avatar) {
				return fmt.Errorf("invalid avatar URL format")
			}
			if len(*req.Avatar) > 500 {
				return fmt.Errorf("avatar URL cannot exceed 500 characters")
			}
		}
	}

	if req.Banner != nil {
		if *req.Banner != "" {
			if !isValidURL(*req.Banner) {
				return fmt.Errorf("invalid banner URL format")
			}
			if len(*req.Banner) > 500 {
				return fmt.Errorf("banner URL cannot exceed 500 characters")
			}
		}
	}

	if req.TagLine != nil {
		if *req.TagLine != "" {
			if len(*req.TagLine) > 500 {
				return fmt.Errorf("tagLine cannot exceed 500 characters")
			}
		}
	}

	if req.WebUrl != nil {
		if *req.WebUrl != "" {
			if !isValidURL(*req.WebUrl) {
				return fmt.Errorf("invalid webUrl format")
			}
			if len(*req.WebUrl) > 500 {
				return fmt.Errorf("webUrl cannot exceed 500 characters")
			}
		}
	}

	if req.CompanyName != nil {
		if *req.CompanyName != "" {
			if len(*req.CompanyName) > 100 {
				return fmt.Errorf("companyName cannot exceed 100 characters")
			}
		}
	}

	if req.Country != nil {
		if *req.Country != "" {
			if len(*req.Country) > 100 {
				return fmt.Errorf("country cannot exceed 100 characters")
			}
		}
	}

	if req.Address != nil {
		if *req.Address != "" {
			if len(*req.Address) > 500 {
				return fmt.Errorf("address cannot exceed 500 characters")
			}
		}
	}

	if req.Phone != nil {
		if *req.Phone != "" {
			if len(*req.Phone) > 20 {
				return fmt.Errorf("phone cannot exceed 20 characters")
			}
		}
	}

	if req.FacebookId != nil {
		if *req.FacebookId != "" {
			if len(*req.FacebookId) > 100 {
				return fmt.Errorf("facebookId cannot exceed 100 characters")
			}
		}
	}

	if req.InstagramId != nil {
		if *req.InstagramId != "" {
			if len(*req.InstagramId) > 100 {
				return fmt.Errorf("instagramId cannot exceed 100 characters")
			}
		}
	}

	if req.TwitterId != nil {
		if *req.TwitterId != "" {
			if len(*req.TwitterId) > 100 {
				return fmt.Errorf("twitterId cannot exceed 100 characters")
			}
		}
	}

	if req.LinkedInId != nil {
		if *req.LinkedInId != "" {
			if len(*req.LinkedInId) > 100 {
				return fmt.Errorf("linkedInId cannot exceed 100 characters")
			}
		}
	}

	if req.AccessUserList != nil {
		if len(req.AccessUserList) > 1000 {
			return fmt.Errorf("maximum 1000 users allowed in access list")
		}
		for i, userID := range req.AccessUserList {
			if strings.TrimSpace(userID) == "" {
				return fmt.Errorf("user ID at index %d cannot be empty", i)
			}
			if len(userID) > 100 {
				return fmt.Errorf("user ID at index %d cannot exceed 100 characters", i)
			}
		}
	}

	if req.Permission != nil && *req.Permission != "" {
		validPermissions := map[string]bool{
			"Public":  true,
			"OnlyMe":  true,
			"Circles": true,
		}
		if !validPermissions[*req.Permission] {
			return fmt.Errorf("permission must be one of: Public, OnlyMe, Circles")
		}
	}

	return nil
}

func ValidateUpdateProfileRequest(req *models.UpdateProfileRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}

	if req.FullName != nil {
		if strings.TrimSpace(*req.FullName) == "" {
			return fmt.Errorf("fullName cannot be empty or whitespace only")
		}
		if len(*req.FullName) > 100 {
			return fmt.Errorf("fullName cannot exceed 100 characters")
		}
	}

	if req.Avatar != nil {
		if *req.Avatar != "" {
			if !isValidURL(*req.Avatar) {
				return fmt.Errorf("invalid avatar URL format")
			}
			if len(*req.Avatar) > 500 {
				return fmt.Errorf("avatar URL cannot exceed 500 characters")
			}
		}
	}

	if req.Banner != nil {
		if *req.Banner != "" {
			if !isValidURL(*req.Banner) {
				return fmt.Errorf("invalid banner URL format")
			}
			if len(*req.Banner) > 500 {
				return fmt.Errorf("banner URL cannot exceed 500 characters")
			}
		}
	}

	if req.TagLine != nil {
		if *req.TagLine != "" {
			if len(*req.TagLine) > 500 {
				return fmt.Errorf("tagLine cannot exceed 500 characters")
			}
		}
	}

	if req.SocialName != nil {
		if *req.SocialName != "" {
			if strings.TrimSpace(*req.SocialName) == "" {
				return fmt.Errorf("socialName cannot be empty or whitespace only")
			}
			if len(*req.SocialName) > 50 {
				return fmt.Errorf("socialName cannot exceed 50 characters")
			}
		}
	}

	return nil
}

func ValidateUpdateLastSeenRequest(req *models.UpdateLastSeenRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}

	if strings.TrimSpace(req.UserId) == "" {
		return fmt.Errorf("userId is required")
	}

	return nil
}

func ValidateProfileQueryFilter(filter *models.ProfileQueryFilter) error {
	if filter == nil {
		return fmt.Errorf("filter is required")
	}

	// Auto-fix pagination defaults
	if filter.Page < 1 {
		filter.Page = 1
	}

	if filter.Limit < 1 {
		filter.Limit = 10
	}

	if filter.Limit > 100 {
		filter.Limit = 100
	}

	if filter.Search != "" {
		if len(filter.Search) > 200 {
			return fmt.Errorf("search term cannot exceed 200 characters")
		}
	}

	return nil
}

func ValidateUserIdList(userIds []string) error {
	// Allow empty arrays - will return empty results (REST best practice)
	if len(userIds) == 0 {
		return nil
	}

	if len(userIds) > 1000 {
		return fmt.Errorf("maximum 1000 user IDs allowed")
	}

	for i, userID := range userIds {
		trimmed := strings.TrimSpace(userID)
		if trimmed == "" {
			return fmt.Errorf("user ID at index %d cannot be empty", i)
		}
		if len(trimmed) > 100 {
			return fmt.Errorf("user ID at index %d cannot exceed 100 characters", i)
		}
		// Validate UUID format (standard format: 8-4-4-4-12 hex digits)
		if !isValidUUIDFormat(trimmed) {
			return fmt.Errorf("user ID at index %d has invalid UUID format: %s", i, trimmed)
		}
	}

	return nil
}

// isValidUUIDFormat checks if a string matches standard UUID format
func isValidUUIDFormat(s string) bool {
	// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if len(s) != 36 {
		return false
	}
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func ValidateIncrementValue(inc int) error {
	if inc < -1000 || inc > 1000 {
		return fmt.Errorf("increment value must be between -1000 and 1000")
	}
	return nil
}

// ValidateSocialName validates social name format with comprehensive rules
// Rules:
// - Must be 3-50 characters long
// - Can only contain alphanumeric characters, underscores, hyphens, and periods
// - Must start and end with alphanumeric character
// - Cannot contain spaces or other special characters
// - Cannot have consecutive special characters (e.g., "..", "__", "--")
func ValidateSocialName(name string) error {
	trimmed := strings.TrimSpace(name)
	
	// Check if empty after trimming
	if trimmed == "" {
		return fmt.Errorf("social name cannot be empty")
	}
	
	// Check if contains spaces (before or after trim - reject both)
	if strings.Contains(name, " ") {
		return fmt.Errorf("social name cannot contain spaces")
	}
	
	// Check length constraints
	if len(trimmed) < socialNameMinLength {
		return fmt.Errorf("social name must be at least %d characters", socialNameMinLength)
	}
	if len(trimmed) > socialNameMaxLength {
		return fmt.Errorf("social name cannot exceed %d characters", socialNameMaxLength)
	}
	
	// Check for consecutive special characters
	if consecutiveSpecialChars.MatchString(trimmed) {
		return fmt.Errorf("social name cannot contain consecutive special characters")
	}
	
	// Check format: alphanumeric, underscore, hyphen, period only
	// Must start and end with alphanumeric
	if !socialNamePattern.MatchString(trimmed) {
		return fmt.Errorf("social name must start and end with alphanumeric characters and can only contain letters, numbers, underscores, hyphens, and periods")
	}
	
	return nil
}

func isValidEmail(email string) bool {
	// Basic check - real validation should use proper regex
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

func isValidURL(urlStr string) bool {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return parsed.Scheme != "" && parsed.Host != ""
}

