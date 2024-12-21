package handlers

import (
	"database/sql"
	"math"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/momokii/simple-chat-app/internal/database"
	"github.com/momokii/simple-chat-app/internal/models"
	"github.com/momokii/simple-chat-app/internal/repository/room"
	"github.com/momokii/simple-chat-app/pkg/utils"
)

type RoomChatHandler struct {
	roomChatRepo room.RoomChatRepo
}

func NewRoomChatHandler(roomChatRepo room.RoomChatRepo) *RoomChatHandler {
	return &RoomChatHandler{
		roomChatRepo: roomChatRepo,
	}
}

func (h *RoomChatHandler) GetRoomList(c *fiber.Ctx) error {
	// user data
	user := c.Locals("user").(models.UserSession)

	// QUERY PARAMS
	// check query if get room list for room created by user it self
	is_user_room := c.Query("self")
	if is_user_room != "true" {
		is_user_room = "false"
	}
	filter := c.Query("filter")
	search := c.Query("search")
	page := c.QueryInt("page")
	if page == 0 {
		page = 1
	}
	per_page := c.QueryInt("per_page")
	if per_page == 0 {
		per_page = 5
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer func() {
		database.CommitOrRollback(tx, c, err)
	}()

	rooms, total, err := h.roomChatRepo.Find(tx, user.Id, page, per_page, is_user_room, filter, search)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to get room list")
	}

	if len(*rooms) == 0 {
		rooms = &[]models.RoomChatDataShow{}
	}

	// count total page
	total_page := int(math.Ceil(float64(total) / float64(per_page)))

	return utils.ResponseWithData(c, fiber.StatusOK, "Success Get Room List", fiber.Map{
		"rooms": rooms,
		"pagination": fiber.Map{
			"current_page": page,
			"per_page":     per_page,
			"total_items":  total,
			"total_page":   total_page,
		},
	})
}

func (h *RoomChatHandler) CreateRoom(c *fiber.Ctx) error {
	// get user data
	user := c.Locals("user").(models.UserSession)

	room := new(models.RoomChatCreate)

	if err := c.BodyParser(room); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request")
	}

	if err := utils.ValidateStruct(room); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Field() {
			case "RoomName":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Room Name must be alphanumeric and between 3-25 characters")
			case "Description":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Description must be alphanumeric and between 6-50 characters")
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

	// first create random code and check it if already exist
	var codeRoom string
	for {
		codeRoom = utils.RandomString(6)

		isRoomExist, err := h.roomChatRepo.FindByCodeOrAndId(tx, codeRoom, 0)
		if err != nil {
			return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to check room code")
		}

		// if id is 0, so the room is not exist/ available
		if isRoomExist.Id == 0 {
			break
		}
	}

	// create new room
	roomData := models.RoomChat{
		RoomCode:    codeRoom,
		CreatedBy:   user.Id,
		RoomName:    room.RoomName,
		Description: room.Description,
	}

	if err := h.roomChatRepo.Create(tx, &roomData); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to create new room")
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "Success Create New Room")
}

func (h *RoomChatHandler) EditRoom(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserSession)

	roomUpdateInput := new(models.RoomChatEdit)
	if err := c.BodyParser(roomUpdateInput); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request")
	}

	if err := utils.ValidateStruct(roomUpdateInput); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Field() {
			case "Id":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Room ID must be numeric and required")
			case "RoomName":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Room Name must be alphanumeric and between 3-25 characters")
			case "Description":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Description must be alphanumeric and between 6-50 characters")
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

	// check if room is exist
	isRoomExist, err := h.roomChatRepo.FindByCodeOrAndId(tx, "", roomUpdateInput.Id)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to check room")
	}

	if err == sql.ErrNoRows {
		return utils.ResponseError(c, fiber.StatusNotFound, "Room not found")
	}

	// if exist, check if user is the creator or not
	if isRoomExist.CreatedBy != user.Id {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "You are not allowed to edit this room")
	}

	// update rrom data
	updateRoom := models.RoomChat{
		Id:          roomUpdateInput.Id,
		RoomName:    roomUpdateInput.RoomName,
		Description: roomUpdateInput.Description,
	}
	if err := h.roomChatRepo.Update(tx, &updateRoom); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to update room")
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "Success Edit Room")
}

func (h *RoomChatHandler) DeleteRoom(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserSession)

	roomDelete := new(models.RoomChatDelete)

	if err := c.BodyParser(roomDelete); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request "+err.Error())
	}

	if err := utils.ValidateStruct(roomDelete); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Field() {
			case "Id":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Room ID must be numeric and required")
			}
		}
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to start transaction: ")
	}
	defer func() {
		database.CommitOrRollback(tx, c, err)
	}()

	// check if room is exist
	isRoomExist, err := h.roomChatRepo.FindByCodeOrAndId(tx, "", roomDelete.Id)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to check room ")
	}

	if err == sql.ErrNoRows {
		return utils.ResponseError(c, fiber.StatusNotFound, "Room not found")
	}

	// if exist, check if user is the creator or not
	if user.Id != isRoomExist.CreatedBy {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "You are not allowed to delete this room")
	}

	// delete room
	if err := h.roomChatRepo.Delete(tx, roomDelete.Id); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to delete room")
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "Success Delete Room")
}
