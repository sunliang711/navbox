package dto

import "time"

type LoginReq struct {
	Username string `json:"username" binding:"omitempty,max=64"`
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
	Authenticated bool         `json:"authenticated"`
	User          *AccountResp `json:"user,omitempty"`
}

type AccountResp struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	Role        string     `json:"role"`
	Disabled    bool       `json:"disabled"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type ActiveSessionResp struct {
	ID         string    `json:"id"`
	ClientIP   string    `json:"client_ip"`
	UserAgent  string    `json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Current    bool      `json:"current"`
}

type CreateAccountReq struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	Username        string `json:"username" binding:"required,min=3,max=64"`
	Password        string `json:"password" binding:"required,min=8,max=72"`
	Role            string `json:"role" binding:"required,oneof=admin viewer"`
}

type UpdateAccountReq struct {
	CurrentPassword string  `json:"current_password" binding:"required"`
	Username        *string `json:"username" binding:"omitempty,min=3,max=64"`
	Role            *string `json:"role" binding:"omitempty,oneof=admin viewer"`
	Disabled        *bool   `json:"disabled"`
	NewPassword     *string `json:"new_password" binding:"omitempty,min=8,max=72"`
}
