package models

import (
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/lib/pq"
)

// Profile represents the complete profile entity in the database
// Updated for relational schema: columns are explicit, arrays for access_user_list
type Profile struct {
	// Primary key - maps to 'user_id' in the new schema
	ObjectId uuid.UUID `json:"objectId" bson:"objectId" db:"user_id"`

	// Core fields
	FullName   string `json:"fullName" bson:"fullName" db:"full_name"`
	SocialName string `json:"socialName" bson:"socialName" db:"social_name"`
	Email      string `json:"email" bson:"email" db:"email"`
	Avatar     string `json:"avatar" bson:"avatar" db:"avatar"`
	Banner     string `json:"banner" bson:"banner" db:"banner"`
	Tagline    string `json:"tagLine" bson:"tagLine" db:"tagline"`

	// Timestamps - both Unix timestamps and TIMESTAMPTZ for compatibility
	CreatedDate int64     `json:"createdDate" bson:"createdDate" db:"created_date"`
	LastUpdated int64     `json:"lastUpdated" bson:"lastUpdated" db:"last_updated"`
	LastSeen    int64     `json:"lastSeen" bson:"lastSeen" db:"last_seen"`
	CreatedAt   time.Time `json:"createdAt,omitempty" bson:"createdAt,omitempty" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty" bson:"updatedAt,omitempty" db:"updated_at"`

	// Optional fields
	Birthday    int64 `json:"birthday" bson:"birthday" db:"birthday"`
	WebUrl      string `json:"webUrl" bson:"webUrl" db:"web_url"`
	CompanyName string `json:"companyName" bson:"companyName" db:"company_name"`
	Country     string `json:"country" bson:"country" db:"country"`
	Address     string `json:"address" bson:"address" db:"address"`
	Phone       string `json:"phone" bson:"phone" db:"phone"`

	// Count fields
	VoteCount     int64 `json:"voteCount" bson:"voteCount" db:"vote_count"`
	ShareCount    int64 `json:"shareCount" bson:"shareCount" db:"share_count"`
	FollowCount   int64 `json:"followCount" bson:"followCount" db:"follow_count"`
	FollowerCount int64 `json:"followerCount" bson:"followerCount" db:"follower_count"`
	PostCount     int64 `json:"postCount" bson:"postCount" db:"post_count"`

	// Social IDs
	FacebookId  string `json:"facebookId" bson:"facebookId" db:"facebook_id"`
	InstagramId string `json:"instagramId" bson:"instagramId" db:"instagram_id"`
	TwitterId   string `json:"twitterId" bson:"twitterId" db:"twitter_id"`
	LinkedInId  string `json:"linkedInId" bson:"linkedInId" db:"linkedin_id"`

	// Access control
	AccessUserList pq.StringArray `json:"accessUserList" bson:"accessUserList" db:"access_user_list"` // Use pq.StringArray for PostgreSQL arrays
	Permission     string         `json:"permission" bson:"permission" db:"permission"`
}

type UpdateLastSeenRequest struct {
	UserId string `json:"userId" validate:"required"`
}

type UpdateProfileRequest struct {
	FullName    *string `json:"fullName,omitempty"`
	Avatar      *string `json:"avatar,omitempty"`
	Banner      *string `json:"banner,omitempty"`
	TagLine     *string `json:"tagLine,omitempty"`
	SocialName  *string `json:"socialName,omitempty"`
	WebUrl      *string `json:"webUrl,omitempty"`
	CompanyName *string `json:"companyName,omitempty"`
	FacebookId  *string `json:"facebookId,omitempty"`
	InstagramId *string `json:"instagramId,omitempty"`
	TwitterId   *string `json:"twitterId,omitempty"`
}

type CreateProfileRequest struct {
	ObjectId         uuid.UUID `json:"objectId" validate:"required"`
	FullName         *string   `json:"fullName,omitempty"`
	SocialName       *string   `json:"socialName,omitempty"`
	Email            *string   `json:"email,omitempty"`
	Avatar           *string   `json:"avatar,omitempty"`
	Banner           *string   `json:"banner,omitempty"`
	TagLine          *string   `json:"tagLine,omitempty"`
	CreatedDate      *int64    `json:"createdDate,omitempty"`
	LastUpdated      *int64    `json:"lastUpdated,omitempty"`
	LastSeen         *int64    `json:"lastSeen,omitempty"`
	Birthday         *int64    `json:"birthday,omitempty"`
	WebUrl           *string   `json:"webUrl,omitempty"`
	CompanyName      *string   `json:"companyName,omitempty"`
	Country          *string   `json:"country,omitempty"`
	Address          *string   `json:"address,omitempty"`
	Phone            *string   `json:"phone,omitempty"`
	VoteCount        *int64    `json:"voteCount,omitempty"`
	ShareCount       *int64    `json:"shareCount,omitempty"`
	FollowCount      *int64    `json:"followCount,omitempty"`
	FollowerCount    *int64    `json:"followerCount,omitempty"`
	PostCount        *int64    `json:"postCount,omitempty"`
	FacebookId       *string   `json:"facebookId,omitempty"`
	InstagramId      *string   `json:"instagramId,omitempty"`
	TwitterId        *string   `json:"twitterId,omitempty"`
	LinkedInId       *string   `json:"linkedInId,omitempty"`
	AccessUserList   []string  `json:"accessUserList,omitempty"`
	Permission       *string   `json:"permission,omitempty"`
}

type ProfileQueryFilter struct {
	Search string `json:"search,omitempty"`
	Page   int64  `json:"page,omitempty"`
	Limit  int64  `json:"limit,omitempty"`
}

type ProfilesResponse struct {
	Profiles []Profile `json:"profiles"`
	Total    int64     `json:"total"`
}



