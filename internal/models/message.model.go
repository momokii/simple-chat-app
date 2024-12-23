package models

type Message struct {
	Id        int    `json:"id" validate:"required"`
	RoomId    int    `json:"room_id" validate:"required"`
	SenderId  int    `json:"sender_id" validate:"required"`
	Content   string `json:"content" validate:"required,min=1,max=140"`
	CreatedAt string `json:"created_at" validate:"required"`
}

type MessageShow struct {
	Id             int    `json:"id" validate:"required"`
	RoomId         int    `json:"room_id" validate:"required"`
	SenderUsername string `json:"sender_username" validate:"required"`
	Content        string `json:"content" validate:"required,min=1,max=140"`
	CreatedAt      string `json:"created_at" validate:"required"`
}

type MessageCreate struct {
	RoomCode string `json:"room_code" validate:"required"`
	SenderId int    `json:"sender_id" validate:"required"`
	Content  string `json:"content" validate:"required,min=1"`
}
