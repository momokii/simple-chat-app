package models

type Session struct {
	Id        int    `json:"id"`
	SessionId string `json:"session_id"`
	UserId    int    `json:"user_id"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

type SessionCreate struct {
	SessionId string `json:"session_id" validate:"required"`
	UserId    int    `json:"user_id" validate:"required"`
	ExpiresAt string `json:"expires_at" validate:"required"`
	CreatedAt string `json:"created_at"`
}
