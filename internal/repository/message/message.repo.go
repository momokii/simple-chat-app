package message

import (
	"database/sql"
	"errors"

	"github.com/momokii/simple-chat-app/internal/models"
)

type MessageRepo struct{}

func NewMessageRepo() *MessageRepo {
	return &MessageRepo{}
}

func (r *MessageRepo) FindByRoom(tx *sql.Tx, roomId int) (*[]models.MessageShow, error) {
	var messages []models.MessageShow

	if roomId < 1 {
		return &messages, errors.New("Room ID is required")
	}

	query := "SELECT m.id, m.room_id, u.username, m.content, m.created_at FROM messages m LEFT JOIN users u ON m.sender_id = u.id WHERE room_id = $1 ORDER BY created_at ASC"

	rows, err := tx.Query(query, roomId)
	if err != nil {
		return &messages, err
	}
	defer rows.Close()

	for rows.Next() {
		var message models.MessageShow

		if err := rows.Scan(&message.Id, &message.RoomId, &message.SenderUsername, &message.Content, &message.CreatedAt); err != nil {
			return &messages, err
		}

		messages = append(messages, message)
	}

	return &messages, nil
}

func (r *MessageRepo) Create(tx *sql.Tx, message *models.Message) error {
	query := "INSERT INTO messages (room_id, sender_id, content, created_at) VALUES ($1, $2, $3, NOW()) RETURNING id"

	if _, err := tx.Exec(query, message.RoomId, message.SenderId, message.Content); err != nil {
		return err
	}

	return nil
}
