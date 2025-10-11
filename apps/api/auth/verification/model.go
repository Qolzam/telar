package verification

type VerifySignupRequest struct {
	Token        string `json:"token"`
	Code         string `json:"code"`
	ResponseType string `json:"responseType"`
}
