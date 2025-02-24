package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/momokii/go-llmbridge/pkg/openai"
	"github.com/momokii/simple-chat-app/internal/database"
	"github.com/momokii/simple-chat-app/internal/models"
	"github.com/momokii/simple-chat-app/internal/repository/room"
	roommember "github.com/momokii/simple-chat-app/internal/repository/room_member"
	"github.com/momokii/simple-chat-app/internal/repository/room_train"
	"github.com/momokii/simple-chat-app/pkg/utils"
	"golang.org/x/crypto/bcrypt"

	sso_models "github.com/momokii/go-sso-web/pkg/models"
	sso_conn_room_reserved "github.com/momokii/go-sso-web/pkg/repository/conn_room_credit_reserved"
	sso_user "github.com/momokii/go-sso-web/pkg/repository/user"
	sso_credit_reserved "github.com/momokii/go-sso-web/pkg/repository/user_credit_reserved"
	sso_utils "github.com/momokii/go-sso-web/pkg/utils"
)

type RoomChatHandler struct {
	roomChatRepo               room.RoomChatRepo
	roomChatTrainRepo          room_train.RoomChatTrainRepo
	roomMemberRepo             roommember.RoomMemberRepo
	openaiClient               openai.OpenAI
	userRepo                   sso_user.UserRepo
	reservedTokenRepo          sso_credit_reserved.UserCreditReserved
	connRoomCreditReservedRepo sso_conn_room_reserved.ConnRoomCreditReserved
}

func NewRoomChatHandler(roomChatRepo room.RoomChatRepo, roomTrainRepo room_train.RoomChatTrainRepo, roomMemberRepo roommember.RoomMemberRepo, openaiClient openai.OpenAI, userRepo sso_user.UserRepo, reservedTokenRepo sso_credit_reserved.UserCreditReserved, connRoomCreditReservedRepo sso_conn_room_reserved.ConnRoomCreditReserved) *RoomChatHandler {
	return &RoomChatHandler{
		roomChatRepo:               roomChatRepo,
		roomChatTrainRepo:          roomTrainRepo,
		roomMemberRepo:             roomMemberRepo,
		openaiClient:               openaiClient,
		userRepo:                   userRepo,
		reservedTokenRepo:          reservedTokenRepo,
		connRoomCreditReservedRepo: connRoomCreditReservedRepo,
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

func (h *RoomChatHandler) RoomTrainChatView(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserSession)

	return c.Render("chatroom_train", fiber.Map{
		"Title": "Chatroom - Chat Nge-Chat",
		"User":  user,
	})
}

func (h *RoomChatHandler) GetTrainRoomData(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserSession)

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

	// check room
	checkRoom, err := h.roomChatRepo.FindByCodeOrAndId(tx, roomCode, 0)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to check room")
	}

	if checkRoom.Id == 0 {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Room is not exist")
	}

	// check if the room is train room, if not r
	if !checkRoom.IsTrainRoom {
		return utils.ResponseError(c, fiber.StatusBadRequest, "This is not train room")
	}

	// check if room is created by user or not, if not return error because only creator can access this page (train page)
	if checkRoom.CreatedBy != user.Id {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "You are not allowed to access this page")
	}

	// get train room detail data
	trainRoomDetail, err := h.roomChatTrainRepo.FindByRoomCode(tx, roomCode)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to get train room data")
	}

	return utils.ResponseWithData(c, fiber.StatusOK, "Success Get Train Room Data", fiber.Map{
		"room":        checkRoom,
		"room_detail": trainRoomDetail,
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

	// if the room is train room, so return error because train room will be on different page
	if roomData.IsTrainRoom {
		return utils.ResponseError(c, fiber.StatusBadRequest, "This is train room, please go to train room page")
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
	is_train_room := c.Query("train_rizz")
	if is_train_room != "true" {
		is_train_room = "false"
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

	rooms, total, err := h.roomChatRepo.Find(tx, user.Id, page, per_page, is_user_room, is_room_joined, is_train_room, filter, search)
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

func (h *RoomChatHandler) CreateTrainRoom(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserSession)

	roomTrain := new(models.RoomChatTrainCreate)

	if err := c.BodyParser(roomTrain); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request")
	}

	if err := utils.ValidateStruct(roomTrain); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Field() {
			case "Gender":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Gender is required")
			case "Language":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Language is required")
			case "RangeAge":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Range Age is required")
			}
		}
	}

	// start tx
	tx, err := database.DB.Begin()
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer func() {
		database.CommitOrRollback(tx, c, err)
	}()

	// check if user has enough credit to create train room
	user_data, err := h.userRepo.FindByID(tx, user.Id)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to get user data")
	}

	if user_data.Id == 0 {
		return utils.ResponseError(c, fiber.StatusBadRequest, "User not found")
	}

	if user_data.CreditToken < utils.FEATURE_DATING_CHAT_SIMULATION_COST {
		return utils.ResponseError(c, fiber.StatusBadRequest, "You don't have enough credit to create this room")
	}

	// start process

	// init message to openai to get description for train room mate using llm
	baseMessageReq := fmt.Sprintf(`"Buat profil *fictional* untuk simulasi dating app (Tinder/Bumble vibe) dengan kriteria yang akan dijelaskan di bawah. 
	Pastikan bahasanya SUPER CASUAL, pakai slang gen Z, emoji, dan deskripsi unik ala bio Instagram/Tinder/Bumble pada umumnya.
	
	Hindari kalimat sangat formal—bayangkan seperti sedang bikin profil buat temen yang sok asik!". 
	
	Data dasar yang dimiliki dan diprovide adalah berikut: 
		Gender: %s 
		Range Age: %s 
		Main Language: %s

	Berdasarkan data di atas, Tambahkan detail dengan poin yang ada di bawah ini dengan disesuaikan dengan data yang diberikan di atas (gender, language, dan range age):
		1. Employment Type, bisa berikan penjelasan bagian ini secara sederhana atau unik juga bisa

		2. Description: 
		- Fokus pada kebiasaan unik & relatable, contoh sebagai referensi (selalu coba untuk membuatnya beda dari contoh diberikan jika memungkinkan): 
			- "Cewek yang bisa nangis nonton Drakor, tapi juga bisa gebukin tikus pake sandal jepit 😤" 
			- "Cowok pecinta kopi hitam & motor tua. Auto ghosting kalo lo bilang 'es kopi susu lebih enak' ☕"
		- Bisa hanya sekadar sederhana, contoh:
			- "Cewek yang suka jalan-jalan"
			- "Cowok yang suka main game"

		3. Hobby: 
		- Pakai format visual + emoji, contoh sebagai referensi (selalu coba untuk membuatnya beda dari contoh diberikan jika memungkinkan): 
			- "Nyari spot aestetik buat feed IG 📸 | Bikin playlist Spotify buat setiap mood (galau, semangat, atau pengen jadi ikan 🐠)" 
			- "Nge-gym... eh, maksudnya foto di gym terus post story 🏋️♂️"
		- Bisa hanya sekadar sederhana, contoh:
			- "Main game"
			- "Nonton film"

		4. Personality: 
		- Gabungkan sifat + kebiasaan random, contoh sebagai referensi (selalu coba untuk membuatnya beda dari contoh diberikan jika memungkinkan): 
			- "Kocak ga jelas tapi bisa deep talk ✨ | Suka marahin diri sendiri kalo lupa nyimpen kunci 🔑" 
			- "Humor sarkas level 100 🗡️ | Auto jadi ibu-ibu kalo liat orang parkir sembarangan 🚗💢"
		- Bisa hanya sekadar sederhana, contoh:
			- "Pluviofile"
			- "Introvert"
	`, roomTrain.Gender, roomTrain.RangeAge, roomTrain.Language)
	baseResponseFormat := openai.OACreateResponseFormat(
		"base_format_response",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"employment_type": map[string]interface{}{"type": "string"},
				"description":     map[string]interface{}{"type": "string"},
				"hobby":           map[string]interface{}{"type": "string"},
				"personality":     map[string]interface{}{"type": "string"},
			},
		},
	)

	initMessage := []openai.OAMessageReq{
		{
			Role:    "user",
			Content: baseMessageReq,
		},
	}

	initResponse, err := h.openaiClient.OpenAIGetFirstContentDataResp(&initMessage, true, &baseResponseFormat, false, nil)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to get initial message response")
	}

	initResData := new(models.RoomChatTrainCreationRes)
	if err := json.Unmarshal([]byte(initResponse.Content), initResData); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to marshal response")
	}

	// first add new room data (basic room data)

	// create code room
	var codeRoom string
	for {
		codeRoom = utils.RandomString(6)

		isRoomExist, err := h.roomChatRepo.FindByCodeOrAndId(tx, codeRoom, 0)
		if err != nil {
			return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to check room code")
		}

		if isRoomExist.Id == 0 {
			break
		}
	}

	newRoom := models.RoomChat{
		CreatedBy:   user.Id,
		RoomName:    "Train Room",
		Description: "-",
		IsTrainRoom: true,
		IsPrivate:   false,
		RoomCode:    codeRoom,
	}

	if err := h.roomChatRepo.Create(tx, &newRoom); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to create new room")
	}

	// add new train room data
	newRoomTrain := models.RoomChatTrain{
		RoomCode:       codeRoom,
		Gender:         roomTrain.Gender,
		Language:       roomTrain.Language,
		RangeAge:       roomTrain.RangeAge,
		EmploymentType: initResData.EmploymentType,
		Description:    initResData.Description,
		Hobby:          initResData.Hobby,
		Personality:    initResData.Personality,
	}

	if err := h.roomChatTrainRepo.Create(tx, &newRoomTrain); err != nil {
		fmt.Println(err)
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to create new train room")
	}

	// add data to reserved token
	reserved_token := sso_models.UserCreditReserved{
		UserId:      user.Id,
		Credit:      utils.FEATURE_DATING_CHAT_SIMULATION_COST,
		FeatureType: "chat-ai", // enum type for chat ai
		Status:      "pending", // enum type for status
	}
	id_reserved, err := h.reservedTokenRepo.Create(tx, &reserved_token)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to create reserved token")
	}

	// if success add data to conn reserved token for chat data to have connection from reserved token to room
	conn_room_reserved_token := sso_models.ConnRoomCreditReserved{
		RoomCode:             codeRoom,
		UserCreditReservedId: id_reserved,
	}
	if err := h.connRoomCreditReservedRepo.Create(tx, &conn_room_reserved_token); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to create conn room credit reserved")
	}

	// deduct token from user data
	if err := sso_utils.UpdateUserCredit(tx, h.userRepo, user_data, utils.FEATURE_DATING_CHAT_SIMULATION_COST); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to deduct user credit")
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "Success Create Train Room")
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
		IsTrainRoom: false,
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
	// process here will be
	// - change the reserved_token status to completed
	// - and delete the room
	// (dont care about detailed room_chat_train because it will be deleted automatically because of foreign key)
	// just check if the room si room_chat_ai or not and if yes executed flow above and if not just delete the room

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

	// if the room is train room, so delete the reserved token and conn room credit reserved
	if isRoomExist.IsTrainRoom {
		// get room credit reserved data
		room_reserved_data, err := h.reservedTokenRepo.FindRoomByCode(tx, isRoomExist.RoomCode)
		if err != nil {
			return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to get room credit reserved data")
		}

		if room_reserved_data.Id == 0 {
			return utils.ResponseError(c, fiber.StatusBadRequest, "Room credit reserved data not found")
		}

		// update the status of reserved token to completed if the room is still active
		if room_reserved_data.IsHaveRoomActive {
			reserved_data_update := sso_models.UserCreditReserved{
				Id:          room_reserved_data.Id,
				Status:      "confirmed", // enum type for status
				Credit:      room_reserved_data.Credit,
				FeatureType: room_reserved_data.FeatureType,
			}
			if err := h.reservedTokenRepo.Update(tx, &reserved_data_update); err != nil {
				return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to update reserved token")
			}
		}

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
