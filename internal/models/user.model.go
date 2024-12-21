package models

type User struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserSession struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
}

type UserChangeUsernameInput struct {
	Id       int    `json:"id"`
	Username string `json:"username" validate:"required,min=5,max=25,alphanum"`
}

type UserChangePasswordInput struct {
	Id          int    `json:"id"`
	Password    string `json:"password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6,max=50,containsany=1234567890,containsany=QWERTYUIOPASDFGHJKLZXCVBNM"`
}
