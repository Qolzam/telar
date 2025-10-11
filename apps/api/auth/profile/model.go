package profile

type ProfileUpdateRequest struct {
	FullName   *string `json:"fullName"`
	Avatar     *string `json:"avatar"`
	Banner     *string `json:"banner"`
	TagLine    *string `json:"tagLine"`
	SocialName *string `json:"socialName"`
}
