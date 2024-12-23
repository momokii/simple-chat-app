package handlers

import (
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/momokii/simple-chat-app/internal/database"
	"github.com/momokii/simple-chat-app/internal/models"
	"github.com/momokii/simple-chat-app/internal/repository/message"
	"github.com/momokii/simple-chat-app/internal/repository/room"
	"github.com/momokii/simple-chat-app/pkg/utils"
)

type MessageHandler struct {
	roomChatRepo room.RoomChatRepo
	message      message.MessageRepo
}

func NewMessageHandler(roomRepo room.RoomChatRepo, messageRepo message.MessageRepo) *MessageHandler {
	return &MessageHandler{
		roomChatRepo: roomRepo,
		message:      messageRepo,
	}
}

func (h *MessageHandler) GetMessageByRoom(c *fiber.Ctx) error {
	roomCode := c.Params("room_code")

	if roomCode == "" {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Room Code is required")
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer func() {
		database.CommitOrRollback(tx, c, err)
	}()

	// check if roomExist
	isRoomExist, err := h.roomChatRepo.FindByCodeOrAndId(tx, roomCode, 0)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to check room")
	}

	if isRoomExist.Id == 0 {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Room is not exist")
	}

	// get message by room
	messages, err := h.message.FindByRoom(tx, isRoomExist.Id)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to get message list")
	}

	if len(*messages) == 0 {
		messages = &[]models.MessageShow{}
	}

	return utils.ResponseWithData(c, fiber.StatusOK, "Success Get Room Message List", fiber.Map{
		"messages": messages,
	})
}

func (h *MessageHandler) SaveNewMessage(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserSession)

	NewMessage := new(models.MessageCreate)
	if err := c.BodyParser(NewMessage); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Failed to parse request body")
	}

	NewMessage.SenderId = user.Id

	if err := utils.ValidateStruct(NewMessage); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Field() {
			case "Content":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Message Content is required")
			case "RoomCode":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Room ID is required")
			}
		}
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer func() {
		database.CommitOrRollback(tx, c, err)
	}()

	// check if room exist
	isRoomExist, err := h.roomChatRepo.FindByCodeOrAndId(tx, NewMessage.RoomCode, 0)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to check room")
	}

	if isRoomExist.Id == 0 {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Room is not exist")
	}

	// save new message
	message := models.Message{
		RoomId:   isRoomExist.Id,
		SenderId: NewMessage.SenderId,
		Content:  NewMessage.Content,
	}
	if err := h.message.Create(tx, &message); err != nil {
		log.Println(err)
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to save new message")
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "Success Save New Message")
}
