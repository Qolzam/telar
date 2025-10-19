package validation

import (
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/profile/models"
)

func TestValidateCreateProfileRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *models.CreateProfileRequest
		wantErr bool
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "valid request with required fields only",
			req: &models.CreateProfileRequest{
				ObjectId: uuid.Must(uuid.NewV4()),
			},
			wantErr: false,
		},
		{
			name: "valid request with all fields",
			req: &models.CreateProfileRequest{
				ObjectId:       uuid.Must(uuid.NewV4()),
				FullName:       stringPtr("John Doe"),
				SocialName:     stringPtr("johndoe"),
				Email:          stringPtr("john@example.com"),
				Avatar:         stringPtr("https://example.com/avatar.jpg"),
				Banner:         stringPtr("https://example.com/banner.jpg"),
				TagLine:        stringPtr("Software Developer"),
				WebUrl:         stringPtr("https://johndoe.com"),
				CompanyName:    stringPtr("Tech Corp"),
				Country:        stringPtr("USA"),
				Address:        stringPtr("123 Main St"),
				Phone:          stringPtr("+1234567890"),
				FacebookId:     stringPtr("johndoe"),
				InstagramId:    stringPtr("johndoe"),
				TwitterId:      stringPtr("johndoe"),
				LinkedInId:     stringPtr("johndoe"),
				AccessUserList: []string{"user1", "user2"},
				Permission:     stringPtr("Public"),
			},
			wantErr: false,
		},
		{
			name: "invalid email format",
			req: &models.CreateProfileRequest{
				ObjectId: uuid.Must(uuid.NewV4()),
				Email:    stringPtr("invalid-email"),
			},
			wantErr: true,
		},
		{
			name: "invalid avatar URL",
			req: &models.CreateProfileRequest{
				ObjectId: uuid.Must(uuid.NewV4()),
				Avatar:   stringPtr("not-a-url"),
			},
			wantErr: true,
		},
		{
			name: "fullName too long",
			req: &models.CreateProfileRequest{
				ObjectId: uuid.Must(uuid.NewV4()),
				FullName: stringPtr(string(make([]byte, 101))),
			},
			wantErr: true,
		},
		{
			name: "invalid permission",
			req: &models.CreateProfileRequest{
				ObjectId:   uuid.Must(uuid.NewV4()),
				Permission: stringPtr("Invalid"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateProfileRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreateProfileRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUpdateProfileRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *models.UpdateProfileRequest
		wantErr bool
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "valid request with no fields",
			req: &models.UpdateProfileRequest{},
			wantErr: false,
		},
		{
			name: "valid request with all fields",
			req: &models.UpdateProfileRequest{
				FullName:   stringPtr("John Doe"),
				Avatar:     stringPtr("https://example.com/avatar.jpg"),
				Banner:     stringPtr("https://example.com/banner.jpg"),
				TagLine:    stringPtr("Software Developer"),
				SocialName: stringPtr("johndoe"),
			},
			wantErr: false,
		},
		{
			name: "fullName empty string",
			req: &models.UpdateProfileRequest{
				FullName: stringPtr(""),
			},
			wantErr: true,
		},
		{
			name: "fullName whitespace only",
			req: &models.UpdateProfileRequest{
				FullName: stringPtr("   "),
			},
			wantErr: true,
		},
		{
			name: "fullName too long",
			req: &models.UpdateProfileRequest{
				FullName: stringPtr(string(make([]byte, 101))),
			},
			wantErr: true,
		},
		{
			name: "invalid avatar URL",
			req: &models.UpdateProfileRequest{
				Avatar: stringPtr("not-a-url"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateProfileRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateProfileRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUpdateLastSeenRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *models.UpdateLastSeenRequest
		wantErr bool
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "valid request",
			req: &models.UpdateLastSeenRequest{
				UserId: "123e4567-e89b-12d3-a456-426614174000",
			},
			wantErr: false,
		},
		{
			name: "empty userId",
			req: &models.UpdateLastSeenRequest{
				UserId: "",
			},
			wantErr: true,
		},
		{
			name: "whitespace userId",
			req: &models.UpdateLastSeenRequest{
				UserId: "   ",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateLastSeenRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateLastSeenRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateProfileQueryFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  *models.ProfileQueryFilter
		wantErr bool
	}{
		{
			name:    "nil filter",
			filter:  nil,
			wantErr: true,
		},
		{
			name: "valid filter with defaults",
			filter: &models.ProfileQueryFilter{},
			wantErr: false,
		},
		{
			name: "valid filter with custom values",
			filter: &models.ProfileQueryFilter{
				Search: "john",
				Page:   2,
				Limit:  20,
			},
			wantErr: false,
		},
		{
			name: "search term too long",
			filter: &models.ProfileQueryFilter{
				Search: string(make([]byte, 201)),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProfileQueryFilter(tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProfileQueryFilter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUserIdList(t *testing.T) {
	validUUID1 := "550e8400-e29b-41d4-a716-446655440000"
	validUUID2 := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	validUUID3 := "6ba7b811-9dad-11d1-80b4-00c04fd430c8"
	
	tests := []struct {
		name     string
		userIds  []string
		wantErr  bool
	}{
		{
			name:     "empty list",
			userIds:  []string{},
			wantErr:  false, // Allow empty arrays (will return empty results)
		},
		{
			name:     "valid list",
			userIds:  []string{validUUID1, validUUID2, validUUID3},
			wantErr:  false,
		},
		{
			name:     "single valid UUID",
			userIds:  []string{validUUID1},
			wantErr:  false,
		},
		{
			name:     "invalid UUID format",
			userIds:  []string{"not-a-uuid"},
			wantErr:  true,
		},
		{
			name:     "mixed valid and invalid",
			userIds:  []string{validUUID1, "invalid", validUUID2},
			wantErr:  true,
		},
		{
			name:     "list too long",
			userIds:  make([]string, 1001),
			wantErr:  true,
		},
		{
			name:     "empty user ID",
			userIds:  []string{validUUID1, "", validUUID2},
			wantErr:  true,
		},
		{
			name:     "whitespace user ID",
			userIds:  []string{validUUID1, "   ", validUUID2},
			wantErr:  true,
		},
		{
			name:     "incomplete UUID",
			userIds:  []string{"550e8400-e29b-41d4"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserIdList(tt.userIds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserIdList() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateIncrementValue(t *testing.T) {
	tests := []struct {
		name    string
		inc     int
		wantErr bool
	}{
		{"valid positive", 100, false},
		{"valid negative", -100, false},
		{"zero", 0, false},
		{"too high positive", 1001, true},
		{"too low negative", -1001, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIncrementValue(tt.inc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIncrementValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSocialName(t *testing.T) {
	tests := []struct {
		name    string
		social  string
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{"valid simple name", "johndoe", false, ""},
		{"valid with numbers", "user123", false, ""},
		{"valid with underscore", "john_doe", false, ""},
		{"valid with hyphen", "john-doe", false, ""},
		{"valid with period", "john.doe", false, ""},
		{"valid mixed", "user_123.test-name", false, ""},
		{"valid minimum length", "abc", false, ""},
		{"valid maximum length", "a" + strings.Repeat("b", 48) + "c", false, ""}, // 50 chars: a + 48*b + c
		
		// Invalid cases - empty/whitespace
		{"empty string", "", true, "empty"},
		{"whitespace only", "   ", true, "empty"},
		
		// Invalid cases - spaces
		{"contains spaces", "john doe", true, "spaces"},
		{"leading space", " johndoe", true, "spaces"},
		{"trailing space", "johndoe ", true, "spaces"},
		
		// Invalid cases - length
		{"too short", "ab", true, "at least"},
		{"too long", "a" + strings.Repeat("b", 49) + "c", true, "exceed"}, // 51 chars
		
		// Invalid cases - special characters
		{"special char @", "user@name", true, "alphanumeric"},
		{"special char #", "user#123", true, "alphanumeric"},
		{"special char !", "user!", true, "alphanumeric"},
		{"emoji", "user😀", true, "alphanumeric"},
		
		// Invalid cases - consecutive special chars
		{"consecutive underscores", "user__name", true, "consecutive"},
		{"consecutive hyphens", "user--name", true, "consecutive"},
		{"consecutive periods", "user..name", true, "consecutive"},
		{"mixed consecutive", "user_-name", true, "consecutive"},
		
		// Invalid cases - start/end with special char
		{"starts with underscore", "_username", true, "start and end"},
		{"starts with hyphen", "-username", true, "start and end"},
		{"starts with period", ".username", true, "start and end"},
		{"ends with underscore", "username_", true, "start and end"},
		{"ends with hyphen", "username-", true, "start and end"},
		{"ends with period", "username.", true, "start and end"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSocialName(tt.social)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSocialName() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			// If we expect an error, check that the error message contains expected text
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateSocialName() error message = %v, should contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}



