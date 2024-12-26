package models

type RoomChat struct {
	Id          int    `json:"id" validate:"required"`
	RoomCode    string `json:"room_code" validate:"required"`
	CreatedBy   int    `json:"created_by" validate:"required"`
	RoomName    string `json:"room_name" validate:"required,min=1,max=30"`
	Description string `json:"description" validate:"required,min=1,max=140"`
	Password    string `json:"password"`
	IsPrivate   bool   `json:"is_private"`
	CreatedAt   string `json:"created_at" validate:"required"`
	UpdatedAt   string `json:"updated_at" validate:"required"`
}

type RoomChatCreate struct {
	RoomName    string `json:"room_name" validate:"required,min=1,max=30"`
	Description string `json:"description" validate:"required,min=1,max=140"`
	Password    string `json:"password" validate:"min=4,max=30,alphanum"`
	IsPrivate   bool   `json:"is_private"`
}

type RoomChatEdit struct {
	RoomChatCreate
	Id        int  `json:"id" validate:"required"`
	OldStatus bool `json:"old_status"`
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
	IsPrivate   bool   `json:"is_private"`
	Password    string `json:"password"`
	CreatedAt   string `json:"created_at" validate:"required"`
}
