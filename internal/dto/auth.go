package dto

type LoginReq struct {
	Password string `json:"password" binding:"required"`
}

type ChangePasswordReq struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

type RestoreStatusResp struct {
	Enabled bool   `json:"enabled"`
	Mode    string `json:"mode"`
}

type RestoreAdminPasswordReq struct {
	RestoreToken string `json:"restore_token" binding:"required"`
	NewPassword  string `json:"new_password" binding:"required,min=8"`
}

type RestoreAdminPasswordResp struct {
	Restored bool `json:"restored"`
}

type SessionResp struct {
	Authenticated bool `json:"authenticated"`
}
