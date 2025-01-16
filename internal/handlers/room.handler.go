package handlers

import (
	"database/sql"
	"math"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/momokii/simple-chat-app/internal/database"
	"github.com/momokii/simple-chat-app/internal/models"
	"github.com/momokii/simple-chat-app/internal/repository/room"
	roommember "github.com/momokii/simple-chat-app/internal/repository/room_member"
	"github.com/momokii/simple-chat-app/pkg/utils"
	"golang.org/x/crypto/bcrypt"
)

type RoomChatHandler struct {
	roomChatRepo   room.RoomChatRepo
	roomMemberRepo roommember.RoomMemberRepo
}

func NewRoomChatHandler(roomChatRepo room.RoomChatRepo, roomMemberRepo roommember.RoomMemberRepo) *RoomChatHandler {
	return &RoomChatHandler{
		roomChatRepo:   roomChatRepo,
		roomMemberRepo: roomMemberRepo,
	}
}

func (h *RoomChatHandler) RoomMainDashboardView(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserSession)

	return c.Render("dashboard", fiber.Map{
		"Title": "Main - Chat Nge-Chat",
		"User":  user,
	})
}

func (h *RoomChatHandler) RoomChatView(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserSession)

	return c.Render("chatroom", fiber.Map{
		"Title": "Chatroom - Chat Nge-Chat",
		"User":  user,
	})
}

func (h *RoomChatHandler) GetRoomData(c *fiber.Ctx) error {
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
	roomData, err := h.roomChatRepo.FindByCodeOrAndId(tx, roomCode, 0)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to check room")
	}

	if roomData.Id == 0 {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Room is not exist")
	}

	// get list member data of the room
	members, err := h.roomMemberRepo.FindByRoom(tx, roomData.Id)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to get room member list")
	}

	if len(*members) == 0 {
		members = &[]models.RoomMemberShow{}
	}

	return utils.ResponseWithData(c, fiber.StatusOK, "Success Get Room Data", fiber.Map{
		"room":    roomData,
		"members": members,
	})
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
	is_room_joined := c.Query("joined")
	if is_room_joined != "true" {
		is_room_joined = "false"
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

	rooms, total, err := h.roomChatRepo.Find(tx, user.Id, page, per_page, is_user_room, is_room_joined, filter, search)
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

			// if private, check password for private room
			if room.IsPrivate {
				switch err.Field() {
				case "Password":
					return utils.ResponseError(c, fiber.StatusBadRequest, "Password is required for private room and must be alphanumeric and between 4-30 characters")
				}
			}

		}
	}

	// if is private and passowrd is empty
	if room.IsPrivate && room.Password == "" {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Password is required for private room")
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

	// if private, hash password
	if room.IsPrivate {
		passwordHashed, err := bcrypt.GenerateFromPassword([]byte(room.Password), 16)
		if err != nil {
			return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to hash password")
		}

		roomData.Password = string(passwordHashed)
		roomData.IsPrivate = true
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

			// check new password error if room is private and the private is change form before is public and change to private
			// or if room is private and password is not empty
			if (roomUpdateInput.IsPrivate && !roomUpdateInput.OldStatus) || (roomUpdateInput.IsPrivate && roomUpdateInput.Password != "") {
				switch err.Field() {
				case "Password":
					return utils.ResponseError(c, fiber.StatusBadRequest, "Password is required for private room and must be alphanumeric and between 4-30 characters")
				}
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
	isUpdatePassword := false // false can happen like when room is public and change to public or private to private and password is empty
	updateRoom := models.RoomChat{
		Id:          roomUpdateInput.Id,
		RoomName:    roomUpdateInput.RoomName,
		Description: roomUpdateInput.Description,
		IsPrivate:   roomUpdateInput.IsPrivate,
	}
	// if the room is private and new data change to public, remove password
	if isRoomExist.IsPrivate && !roomUpdateInput.IsPrivate {
		updateRoom.Password = ""
		isUpdatePassword = true

	} else if (isRoomExist.IsPrivate && roomUpdateInput.IsPrivate && roomUpdateInput.Password != "") || (!isRoomExist.IsPrivate && roomUpdateInput.IsPrivate && roomUpdateInput.Password != "") {
		// if the room is private and password is not empty, hash the password

		newHashedPass, err := bcrypt.GenerateFromPassword([]byte(roomUpdateInput.Password), 16)
		if err != nil {
			return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to hash password")
		}

		updateRoom.Password = string(newHashedPass)
		isUpdatePassword = true
	}

	if err := h.roomChatRepo.Update(tx, &updateRoom, isUpdatePassword); err != nil {
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

// --- below for room member related function ---
func (h *RoomChatHandler) AddJoinRoom(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserSession)

	memberInput := new(models.RoomMemberCreate)
	if err := c.BodyParser(memberInput); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request")
	}

	if err := utils.ValidateStruct(memberInput); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Field() {
			case "RoomId":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Room ID must be numeric and required")
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
	roomCheck, err := h.roomChatRepo.FindByCodeOrAndId(tx, "", memberInput.RoomId)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to check room")
	}

	if roomCheck.Id == 0 {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Room is not exist")
	}

	// check if user is already in the room
	exist, err := h.roomMemberRepo.FindUserInRoom(tx, user.Id, memberInput.RoomId)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to check user in room")
	}

	// if not exist so add user to the room
	if !exist {

		// check the room is private or not, if private check the password and if not private, just add user to the room
		if roomCheck.IsPrivate {
			if memberInput.Password == "" {
				return utils.ResponseError(c, fiber.StatusBadRequest, "Password is required for private room")
			}

			if err := bcrypt.CompareHashAndPassword([]byte(roomCheck.Password), []byte(memberInput.Password)); err != nil {
				return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid password")
			}
		}

		newMember := models.RoomMember{
			RoomId: memberInput.RoomId,
			UserId: user.Id,
		}

		if err := h.roomMemberRepo.Create(tx, &newMember); err != nil {
			return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to join room")
		}
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "Success Join Room")
}

func (h *RoomChatHandler) RemoveRoomMember(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserSession)

	delInput := new(models.RoomMemberCreate)
	if err := c.BodyParser(delInput); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request")
	}

	if err := utils.ValidateStruct(delInput); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Field() {
			case "RoomId":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Room ID must be numeric and required")
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
	roomCheck, err := h.roomChatRepo.FindByCodeOrAndId(tx, "", delInput.RoomId)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to check room")
	}

	if roomCheck.Id == 0 {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Room is not exist")
	}

	// delete user from room member
	if err := h.roomMemberRepo.Delete(tx, user.Id, delInput.RoomId); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to remove room member")
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "Success Remove Room Member")
}
