package room_train

import (
	"database/sql"

	"github.com/momokii/simple-chat-app/internal/models"
)

type RoomChatTrainRepo struct{}

func NewRoomChatTrainRepo() *RoomChatTrainRepo {
	return &RoomChatTrainRepo{}
}

func (r *RoomChatTrainRepo) FindByRoomCode(tx *sql.Tx, roomCode string) (*models.RoomChatTrain, error) {
	var roomTrain models.RoomChatTrain
	roomTrain.RoomCode = roomCode

	query := "SELECT id, gender, language, range_age, employment_type, description, hobby, personality, is_still_continue FROM room_chat_train WHERE room_code = $1"

	if err := tx.QueryRow(query, roomCode).Scan(&roomTrain.Id, &roomTrain.Gender, &roomTrain.Language, &roomTrain.RangeAge, &roomTrain.EmploymentType, &roomTrain.Description, &roomTrain.Hobby, &roomTrain.Personality, &roomTrain.IsStillContinue); err != nil {
		return nil, err
	}

	return &roomTrain, nil
}

func (r *RoomChatTrainRepo) Create(tx *sql.Tx, roomTrain *models.RoomChatTrain) error {
	query := `INSERT INTO room_chat_train (room_code, gender, language, range_age, employment_type, description, hobby, personality) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	if _, err := tx.Exec(query, roomTrain.RoomCode, roomTrain.Gender, roomTrain.Language, roomTrain.RangeAge, roomTrain.EmploymentType, roomTrain.Description, roomTrain.Hobby, roomTrain.Personality); err != nil {
		return err
	}

	return nil
}

func (r *RoomChatTrainRepo) UpdateStatus(tx *sql.Tx, roomCode string) error {
	status := "FALSE"
	query := "UPDATE room_chat_train SET is_still_continue = $1 WHERE room_code = $2"

	if _, err := tx.Exec(query, status, roomCode); err != nil {
		return err
	}

	return nil
}
