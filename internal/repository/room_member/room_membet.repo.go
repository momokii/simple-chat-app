package roommember

import (
	"database/sql"

	"github.com/momokii/simple-chat-app/internal/models"
)

type RoomMemberRepo struct{}

func NewRoomMember() *RoomMemberRepo {
	return &RoomMemberRepo{}
}

func (r *RoomMemberRepo) FindUserInRoom(tx *sql.Tx, userId, roomId int) (bool, error) {
	query := "SELECT COUNT(id) FROM room_members WHERE user_id = $1 AND room_id = $2"

	var count int
	if err := tx.QueryRow(query, userId, roomId).Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *RoomMemberRepo) FindByRoom(tx *sql.Tx, roomId int) (*[]models.RoomMemberShow, error) {
	var members []models.RoomMemberShow

	query := "SELECT rm.id, rm.room_id, rm.user_id, u.username, rm.created_at FROM room_members rm LEFT JOIN users u ON rm.user_id = u.id WHERE room_id = $1 ORDER BY rm.created_at DESC"

	rows, err := tx.Query(query, roomId)
	if err != nil {
		return &members, err
	}
	defer rows.Close()

	for rows.Next() {
		var member models.RoomMemberShow

		if err := rows.Scan(&member.Id, &member.RoomId, &member.UserId, &member.Username, &member.CreatedAt); err != nil {
			return &members, err
		}

		members = append(members, member)
	}

	return &members, nil
}

func (r *RoomMemberRepo) Create(tx *sql.Tx, member *models.RoomMember) error {
	query := "INSERT INTO room_members (room_id, user_id, created_at) VALUES ($1, $2, NOW())"

	if _, err := tx.Exec(query, member.RoomId, member.UserId); err != nil {
		return err
	}

	return nil
}

func (r *RoomMemberRepo) Delete(tx *sql.Tx, userId, roomId int) error {
	query := "DELETE FROM room_members WHERE user_id = $1 AND room_id = $2"

	if _, err := tx.Exec(query, userId, roomId); err != nil {
		return err
	}

	return nil
}
