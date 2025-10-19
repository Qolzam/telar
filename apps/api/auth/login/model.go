package login

type LoginModel struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	ResponseType string `json:"responseType"`
	State        string `json:"state"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
