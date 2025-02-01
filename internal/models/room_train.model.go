package models

import "github.com/momokii/go-llmbridge/pkg/openai"

type RoomChatTrain struct {
	Id              int    `json:"id" validate:"required"`
	RoomCode        string `json:"room_code" validate:"required"`
	Gender          string `json:"gender" validate:"required"`
	Language        string `json:"language" validate:"required"`
	RangeAge        string `json:"range_age" validate:"required"`
	EmploymentType  string `json:"employment_type" validate:"required"`
	Description     string `json:"description" validate:"required"`
	Hobby           string `json:"hobby" validate:"required"`
	Personality     string `json:"personality" validate:"required"`
	IsStillContinue bool   `json:"is_still_continue" validate:"required"`
}

type RoomChatTrainCreationRes struct {
	EmploymentType string `json:"employment_type" validate:"required"`
	Description    string `json:"description" validate:"required"`
	Hobby          string `json:"hobby" validate:"required"`
	Personality    string `json:"personality" validate:"required"`
}

type RoomChatTrainCreate struct {
	Gender   string `json:"gender" validate:"required"`
	RangeAge string `json:"range_age" validate:"required"`
	Language string `json:"language" validate:"required"`
}

type SendMessageLLMReq struct {
	TrainerData RoomChatTrain         `json:"trainer_data" validate:"required"`
	Messages    []openai.OAMessageReq `json:"messages" validate:"required"`
}

type SendMessageLLMRes struct {
	ContinueChat bool   `json:"continue_chat"`
	Content      string `json:"content"`
}
