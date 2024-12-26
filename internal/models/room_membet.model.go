package models

type RoomMember struct {
	Id        int    `json:"id" validate:"required"`
	RoomId    int    `json:"room_id" validate:"required"`
	UserId    int    `json:"user_id" validate:"required"`
	CreatedAt string `json:"created_at" validate:"required"`
}

type RoomMemberShow struct {
	RoomMember
	Username string `json:"username" validate:"required"`
}

type RoomMemberCreate struct {
	RoomId int `json:"room_id" validate:"required"`
	// just case need password field because we need password for the private room chat
	Password string `json:"password" validate:"min=4,max=30"`
}
