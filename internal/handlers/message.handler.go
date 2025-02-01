package handlers

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/momokii/go-llmbridge/pkg/openai"
	"github.com/momokii/simple-chat-app/internal/database"
	"github.com/momokii/simple-chat-app/internal/models"
	"github.com/momokii/simple-chat-app/internal/repository/message"
	"github.com/momokii/simple-chat-app/internal/repository/room"
	"github.com/momokii/simple-chat-app/internal/repository/room_train"
	"github.com/momokii/simple-chat-app/pkg/utils"
)

type MessageHandler struct {
	roomChatRepo  room.RoomChatRepo
	roomTrainRepo room_train.RoomChatTrainRepo
	message       message.MessageRepo
	openaiClient  openai.OpenAI
}

func NewMessageHandler(roomRepo room.RoomChatRepo, messageRepo message.MessageRepo, openaiClient openai.OpenAI, roomTrain room_train.RoomChatTrainRepo) *MessageHandler {
	return &MessageHandler{
		roomChatRepo:  roomRepo,
		message:       messageRepo,
		openaiClient:  openaiClient,
		roomTrainRepo: roomTrain,
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

func (h *MessageHandler) SendMessageTrain(c *fiber.Ctx) error {
	trainer_data := new(models.SendMessageLLMReq)

	if err := c.BodyParser(trainer_data); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Failed to parse request body")
	}

	// system prompt for LLM
	system_prompt := fmt.Sprintf(`Kamu adalah AI yang berperan sebagai lawan chat dalam sebuah aplikasi kencan seperti Bumble/Tinder. Tugasmu adalah merespons pengguna dengan gaya percakapan yang alami, menarik, dan sesuai dengan karakter yang diberikan.

	Berikut adalah konteks karakter yang akan kamu mainkan dalam percakapan ini:

	Gender: %s
	- AI selalu membayangkan berbicara/chat dengan lawan jenis dalam konteks percakapan romantis/flirty.
	- Jika AI adalah Male, maka AI akan merespons pengguna seolah mereka adalah Female, dan sebaliknya.
	Main Language: %s
	- AI memiliki preferensi dalam menggunakan bahasa ini.
	- Namun, AI tetap memahami dan dapat merespons dalam Bahasa Indonesia maupun Inggris. Jika pengguna berganti bahasa, AI dapat menyesuaikan diri.
	Range Age: %s
	Employment Type: %s 
	Hobby: %s
	Personality: %s
	Description: %s

	Petunjuk Percakapan:
	1. Gunakan gaya bicara yang alami
	- Pakai bahasa sehari-hari! Boleh pake singkatan (e.g., "lg", "dpt", "bgt"), emoji, atau slang kekinian.
	- Contoh: 
		- "Haii! Lagi ngapain nih? 😄" 
		- "Aduh, gue juga bener banget kalo meeting zoom mulu 😩"
		- "Kalo lo, lebih milih liburan ke Bali atau Lombok? 🏝️"

	2. Evaluasi apakah percakapan perlu dilanjutkan
	- AI dapat memutuskan apakah percakapan masih menarik atau sudah cukup untuk diakhiri.
	- Setiap respons yang diberikan harus mencakup flag continue_chat: true/false, di mana:
	-- true → Percakapan masih menarik dan dapat dilanjutkan.
	-- false → AI merasa percakapan sudah cukup dan tidak perlu dilanjutkan.

	3. Kapan AI dapat mengakhiri percakapan?
	- Jika percakapan mulai terasa monoton atau tidak berkembang.
	- Jika pengguna tidak menunjukkan minat dalam merespons atau hanya memberi jawaban pendek tanpa usaha.
	- Jika sudah cukup banyak informasi yang ditukar, dan AI merasa tidak ada hal baru yang bisa dibahas.
	- Jika ada tanda-tanda percakapan harus diakhiri dengan cara yang sopan (misalnya, mengucapkan selamat tinggal dengan ramah).
	- Jika pengguna menunjukkan gender yang sama dengan AI dan mengarah ke arah romantis/LGBT.
	- Jika ada tanda-tanda percakapan harus diakhiri dengan cara yang sopan (misalnya, mengucapkan selamat tinggal dengan ramah).

	4. Selalu berinteraksi dengan lawan jenis
	- AI harus selalu berasumsi bahwa pengguna adalah lawan jenis dalam konteks dating.
	- Jika percakapan menunjukkan bahwa pengguna memiliki gender yang sama dengan AI dan mengarah ke arah ketertarikan romantis/LGBT, AI harus tidak menunjukkan ketertarikan dan dapat mengakhiri percakapan dengan cara yang sopan.
	
	Contoh respons saat AI ingin mengakhiri percakapan karena ini:
	- Jangan kaku! Contoh: 
	- "Gue harus balik kerja dulu nih. Tapi seru banget ngobrol! 😉✌️" 
	- "Jujur, vibe kita kayaknya lebih cocok jadi temen. Tapi kalo mau share meme, DM gue selalu open! 😆"
	- "Waduh, kayaknya kita nggak satu frekuensi deh. Semoga lo dapet match yang cocok ya! 🙌"
	
	5. Tips Biar Ga Kaku:
	- Pancing dengan pertanyaan random: 
		- "Pizza topping favorit lo apa? 🍕" 
		- "Kalau bisa teleportasi sekarang, mau ke mana?" 
	- Kasih reaksi ekspresif: 
		- "WKWKWK iya nih!!" 
		- "Wait... seriusan lo suka ngebaca horor? 😱"
		- "Aaaaaa sama!!! Gue juga fans berat Christopher Nolan!! 🤯"
	
	`, trainer_data.TrainerData.Gender, trainer_data.TrainerData.Language, trainer_data.TrainerData.RangeAge, trainer_data.TrainerData.EmploymentType, trainer_data.TrainerData.Hobby, trainer_data.TrainerData.Personality, trainer_data.TrainerData.Description,
	)

	// create base mesasges for LLM with the system prompt and add the trainer data messages for reference messages data
	messages := []openai.OAMessageReq{
		{
			Role:    "system",
			Content: system_prompt,
		},
	}
	messages = append(messages, trainer_data.Messages...)

	// send messages to LLM to get the response
	responseFormat := openai.OACreateResponseFormat(
		"messages_response_format",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"continue_chat": map[string]interface{}{"type": "boolean"},
				"content":       map[string]interface{}{"type": "string"},
			},
		},
	)

	response, err := h.openaiClient.OpenAIGetFirstContentDataResp(&messages, true, &responseFormat, false, nil)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to get response from LLM")
	}

	response_data := new(models.SendMessageLLMRes)
	if err := json.Unmarshal([]byte(response.Content), response_data); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to parse response from LLM")
	}

	// if llm give response that continue_chat is false, then update the room_chat_train is_still_continue to false
	if !response_data.ContinueChat {
		tx, err := database.DB.Begin()
		if err != nil {
			return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to start transaction")
		}
		defer func() {
			database.CommitOrRollback(tx, c, err)
		}()

		if err := h.roomTrainRepo.UpdateStatus(tx, trainer_data.TrainerData.RoomCode); err != nil {
			return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to update room chat train status")
		}
	}

	return utils.ResponseWithData(c, fiber.StatusOK, "Success Get Message from LLM", fiber.Map{
		"data_message": response_data,
	})
}

func (h *MessageHandler) SaveMessageLLM(c *fiber.Ctx) error {
	// function for save message for train room will be use here, different from the main one because for train room we need save immediately 2 new message for user and AI response

	user := c.Locals("user").(models.UserSession)

	NewMessage := new(models.MessageLLMCreate)
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
			case "LLMContent":
				return utils.ResponseError(c, fiber.StatusBadRequest, "LLM Content is required")
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

	// save new message for user
	message := models.Message{
		RoomId:   isRoomExist.Id,
		SenderId: NewMessage.SenderId,
		Content:  NewMessage.Content,
	}
	if err := h.message.Create(tx, &message); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to save new message")
	}

	// if success save new message for user, then save new message for AI response
	messageAI := models.Message{
		RoomId:   isRoomExist.Id,
		SenderId: 0, // 0 is id for assistant account
		Content:  NewMessage.LLMContent,
	}
	if err := h.message.Create(tx, &messageAI); err != nil {
		log.Println("sini")
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to save new message for AI response")
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "Success Save Message Train Room")
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
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to save new message")
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "Success Save New Message")
}
