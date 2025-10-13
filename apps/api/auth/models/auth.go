package models

import (
	"github.com/gofrs/uuid"
)

// UserAuth represents a user authentication record
type UserAuth struct {
	ObjectId      uuid.UUID `json:"objectId" bson:"objectId"`
	Username      string    `json:"username" bson:"username"`
	Password      []byte    `json:"password" bson:"password"`
	Role          string    `json:"role" bson:"role"`
	EmailVerified bool      `json:"emailVerified" bson:"emailVerified"`
	PhoneVerified bool      `json:"phoneVerified" bson:"phoneVerified"`
	CreatedDate   int64     `json:"createdDate" bson:"createdDate"`
	LastUpdated   int64     `json:"lastUpdated" bson:"lastUpdated"`
}

// UserProfile represents a user profile record
type UserProfile struct {
	ObjectId    uuid.UUID `json:"objectId" bson:"objectId"`
	FullName    string    `json:"fullName" bson:"fullName"`
	SocialName  string    `json:"socialName" bson:"socialName"`
	Email       string    `json:"email" bson:"email"`
	Avatar      string    `json:"avatar" bson:"avatar"`
	Banner      string    `json:"banner" bson:"banner"`
	TagLine     string    `json:"tagLine" bson:"tagLine"`
	CreatedDate int64     `json:"createdDate" bson:"createdDate"`
	LastUpdated int64     `json:"lastUpdated" bson:"lastUpdated"`
}

// UserVerification represents a user verification record
type UserVerification struct {
	ObjectId        uuid.UUID `json:"objectId" bson:"objectId"`
	UserId          uuid.UUID `json:"userId" bson:"userId"`
	Code            string    `json:"code" bson:"code"`
	Target          string    `json:"target" bson:"target"`
	TargetType      string    `json:"targetType" bson:"targetType"`
	Counter         int64     `json:"counter" bson:"counter"`
	CreatedDate     int64     `json:"createdDate" bson:"createdDate"`
	RemoteIpAddress string    `json:"remoteIpAddress" bson:"remoteIpAddress"`
	IsVerified      bool      `json:"isVerified" bson:"isVerified"`
	LastUpdated     int64     `json:"lastUpdated" bson:"lastUpdated"`
	// New fields for secure verification flow
	HashedPassword []byte `json:"hashedPassword" bson:"hashedPassword"`
	ExpiresAt      int64  `json:"expiresAt" bson:"expiresAt"`
	Used           bool   `json:"used" bson:"used"`
	// Fix: Store original full name from signup form
	FullName string `json:"fullName" bson:"fullName"`
}

// ProfileUpdate represents profile update data
type ProfileUpdate struct {
	FullName   *string `json:"fullName,omitempty" bson:"fullName,omitempty"`
	SocialName *string `json:"socialName,omitempty" bson:"socialName,omitempty"`
	Avatar     *string `json:"avatar,omitempty" bson:"avatar,omitempty"`
	Banner     *string `json:"banner,omitempty" bson:"banner,omitempty"`
	TagLine    *string `json:"tagLine,omitempty" bson:"tagLine,omitempty"`
}

// ProfileSearchFilter represents profile search filter
type ProfileSearchFilter struct {
	Query         string                 `json:"query" bson:"query"`
	Role          *string                `json:"role,omitempty" bson:"role,omitempty"`
	Verified      *bool                  `json:"verified,omitempty" bson:"verified,omitempty"`
	CreatedAfter  *int64                 `json:"createdAfter,omitempty" bson:"createdAfter,omitempty"`
	CreatedBefore *int64                 `json:"createdBefore,omitempty" bson:"createdBefore,omitempty"`
	Custom        map[string]interface{} `json:"custom,omitempty" bson:"custom,omitempty"`
}

// DatabaseFilter represents a generic database filter
type DatabaseFilter struct {
	ObjectId      *uuid.UUID             `json:"objectId,omitempty" bson:"objectId,omitempty"`
	UserId        *uuid.UUID             `json:"userId,omitempty" bson:"userId,omitempty"`
	Username      *string                `json:"username,omitempty" bson:"username,omitempty"`
	Role          *string                `json:"role,omitempty" bson:"role,omitempty"`
	Email         *string                `json:"email,omitempty" bson:"email,omitempty"`
	Verified      *bool                  `json:"verified,omitempty" bson:"verified,omitempty"`
	CreatedAfter  *int64                 `json:"createdAfter,omitempty" bson:"createdAfter,omitempty"`
	CreatedBefore *int64                 `json:"createdBefore,omitempty" bson:"createdBefore,omitempty"`
	Custom        map[string]interface{} `json:"custom,omitempty" bson:"custom,omitempty"`
}

// DatabaseUpdate represents a generic database update
type DatabaseUpdate struct {
	Set    map[string]interface{} `json:"$set,omitempty" bson:"$set,omitempty"`
	Unset  map[string]interface{} `json:"$unset,omitempty" bson:"$unset,omitempty"`
	Inc    map[string]interface{} `json:"$inc,omitempty" bson:"$inc,omitempty"`
	Push   map[string]interface{} `json:"$push,omitempty" bson:"$push,omitempty"`
	Pull   map[string]interface{} `json:"$pull,omitempty" bson:"$pull,omitempty"`
	Custom map[string]interface{} `json:"custom,omitempty" bson:"custom,omitempty"`
}

// TokenClaim represents JWT token claims
type TokenClaim struct {
	DisplayName string      `json:"displayName" bson:"displayName"`
	SocialName  string      `json:"socialName" bson:"socialName"`
	Email       string      `json:"email" bson:"email"`
	UID         string      `json:"uid" bson:"uid"`
	Role        string      `json:"role" bson:"role"`
	CreatedDate int64       `json:"createdDate" bson:"createdDate"`
	Custom      interface{} `json:"custom,omitempty" bson:"custom,omitempty"`
}

// ProfileInfo represents profile information for tokens
type ProfileInfo struct {
	ID       string `json:"id" bson:"id"`
	Login    string `json:"login" bson:"login"`
	Name     string `json:"name" bson:"name"`
	Audience string `json:"audience" bson:"audience"`
}

// OAuthProvider represents OAuth provider information
type OAuthProvider struct {
	Name         string `json:"name" bson:"name"`
	ProviderId   string `json:"providerId" bson:"providerId"`
	AccessToken  string `json:"accessToken" bson:"accessToken"`
	RefreshToken string `json:"refreshToken" bson:"refreshToken"`
	ExpiresAt    int64  `json:"expiresAt" bson:"expiresAt"`
}

// SocialProfile represents social media profile information
type SocialProfile struct {
	Platform   string `json:"platform" bson:"platform"`
	Username   string `json:"username" bson:"username"`
	ProfileUrl string `json:"profileUrl" bson:"profileUrl"`
	Verified   bool   `json:"verified" bson:"verified"`
}

// ResetToken represents password reset token information
type ResetToken struct {
	Token     string `json:"token" bson:"token"`
	Email     string `json:"email" bson:"email"`
	ExpiresAt int64  `json:"expiresAt" bson:"expiresAt"`
	Used      bool   `json:"used" bson:"used"`
}

// VerificationRequest represents verification request data
type VerificationRequest struct {
	UserId          uuid.UUID `json:"userId" bson:"userId"`
	Target          string    `json:"target" bson:"target"`
	TargetType      string    `json:"targetType" bson:"targetType"`
	RemoteIpAddress string    `json:"remoteIpAddress" bson:"remoteIpAddress"`
	Code            string    `json:"code" bson:"code"`
}

// AuthenticationResult represents authentication result
type AuthenticationResult struct {
	Success   bool         `json:"success" bson:"success"`
	User      *UserAuth    `json:"user,omitempty" bson:"user,omitempty"`
	Profile   *UserProfile `json:"profile,omitempty" bson:"profile,omitempty"`
	Token     string       `json:"token,omitempty" bson:"token,omitempty"`
	Message   string       `json:"message,omitempty" bson:"message,omitempty"`
	ErrorCode string       `json:"errorCode,omitempty" bson:"errorCode,omitempty"`
}
