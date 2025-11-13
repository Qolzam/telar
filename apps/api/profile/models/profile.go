package models

import (
    "github.com/gofrs/uuid"
)

type Profile struct {
	ObjectId         uuid.UUID `json:"objectId" bson:"objectId" db:"objectId"`
	FullName         string    `json:"fullName" bson:"fullName" db:"fullName"`
	SocialName       string    `json:"socialName" bson:"socialName" db:"socialName"`
	Email            string    `json:"email" bson:"email" db:"email"`
	Avatar           string    `json:"avatar" bson:"avatar" db:"avatar"`
	Banner           string    `json:"banner" bson:"banner" db:"banner"`
	TagLine          string    `json:"tagLine" bson:"tagLine" db:"tagLine"`
	CreatedDate      int64     `json:"createdDate" bson:"createdDate" db:"createdDate"`
	LastUpdated      int64     `json:"lastUpdated" bson:"lastUpdated" db:"lastUpdated"`
	LastSeen         int64     `json:"lastSeen" bson:"lastSeen" db:"lastSeen"`
	Birthday         int64     `json:"birthday" bson:"birthday" db:"birthday"`
	WebUrl           string    `json:"webUrl" bson:"webUrl" db:"webUrl"`
	CompanyName      string    `json:"companyName" bson:"companyName" db:"companyName"`
	Country          string    `json:"country" bson:"country" db:"country"`
	Address          string    `json:"address" bson:"address" db:"address"`
	Phone            string    `json:"phone" bson:"phone" db:"phone"`
	VoteCount        int64     `json:"voteCount" bson:"voteCount" db:"voteCount"`
	ShareCount       int64     `json:"shareCount" bson:"shareCount" db:"shareCount"`
	FollowCount      int64     `json:"followCount" bson:"followCount" db:"followCount"`
	FollowerCount    int64     `json:"followerCount" bson:"followerCount" db:"followerCount"`
	PostCount        int64     `json:"postCount" bson:"postCount" db:"postCount"`
	FacebookId       string    `json:"facebookId" bson:"facebookId" db:"facebookId"`
	InstagramId      string    `json:"instagramId" bson:"instagramId" db:"instagramId"`
	TwitterId        string    `json:"twitterId" bson:"twitterId" db:"twitterId"`
	LinkedInId       string    `json:"linkedInId" bson:"linkedInId" db:"linkedInId"`
	AccessUserList   []string  `json:"accessUserList" bson:"accessUserList" db:"accessUserList"`
	Permission       string    `json:"permission" bson:"permission" db:"permission"`
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



