package model

type UserRegisterRequest struct {
	Username string `validate:"required|minLen:4|maxLen:22" json:"username"`
	Password string `validate:"required|minLen:5|maxLen:20" json:"password"`
}

type UserLoginRequest struct {
	Username string `validate:"required|minLen:4|maxLen:22" json:"username"`
	Password string `validate:"required|minLen:5|maxLen:20" json:"password"`
}

type UserInfoResponse struct {
	Id       string `json:"id"`
	Username string `json:"username"`
}

type AllUserInfoResponse struct {
	Id       string `json:"id"`
	Username string `json:"username"`
}
