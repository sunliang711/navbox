package dto

type LoginReq struct {
	Password string `json:"password" binding:"required"`
}

type ChangePasswordReq struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

type SessionResp struct {
	Authenticated bool `json:"authenticated"`
}
