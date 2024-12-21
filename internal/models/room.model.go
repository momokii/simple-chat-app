package models

type RoomChat struct {
	Id          int    `json:"id" validate:"required"`
	RoomCode    string `json:"room_code" validate:"required"`
	CreatedBy   int    `json:"created_by" validate:"required"`
	RoomName    string `json:"room_name" validate:"required,min=1,max=30"`
	Description string `json:"description" validate:"required,min=1,max=140"`
	CreatedAt   string `json:"created_at" validate:"required"`
	UpdatedAt   string `json:"updated_at" validate:"required"`
}

type RoomChatCreate struct {
	RoomName    string `json:"room_name" validate:"required,min=1,max=30"`
	Description string `json:"description" validate:"required,min=1,max=140"`
}

type RoomChatEdit struct {
	RoomChatCreate
	Id int `json:"id" validate:"required"`
}

type RoomChatDelete struct {
	Id int `json:"id" validate:"required"`
}

type RoomChatDataShow struct {
	Id          int    `json:"id" validate:"required"`
	RoomCode    string `json:"room_code" validate:"required"`
	CreatedBy   int    `json:"created_by" validate:"required"`
	Username    string `json:"username" validate:"required"`
	RoomName    string `json:"room_name" validate:"required,min=1,max=30"`
	Description string `json:"description" validate:"required,min=1,max=140"`
	CreatedAt   string `json:"created_at" validate:"required"`
}
