package room

import (
	"database/sql"
	"fmt"

	"github.com/momokii/simple-chat-app/internal/models"
)

type RoomChatRepo struct{}

func NewRoomChatRepo() *RoomChatRepo {
	return &RoomChatRepo{}
}

func (r *RoomChatRepo) Find(tx *sql.Tx, user_id, page, per_page int, is_user_room, is_room_joined, is_room_train, filter, search string) (*[]models.RoomChatDataShow, int, error) {
	var rooms []models.RoomChatDataShow
	var filterType string
	offset := (page - 1) * per_page
	total := 0

	total_query := "SELECT COUNT(rc.id) FROM room_chat rc LEFT JOIN users u ON rc.created_by = u.id WHERE 1 = 1"
	query := "SELECT rc.id, rc.code, rc.created_by, u.username, rc.name, rc.description, rc.created_at, rc.is_private, rc.is_train_room FROM room_chat rc LEFT JOIN users u ON rc.created_by = u.id WHERE 1 = 1"

	idxParam := 1
	paramData := []interface{}{}
	if is_user_room == "true" || is_room_train == "true" {
		// for "my room" and "train room" we just show/retrieve the room that created by the user
		if user_id < 1 {
			return &rooms, 0, fmt.Errorf("User ID is required")
		}
		addQuery := " AND rc.created_by = $" + fmt.Sprint(idxParam)

		total_query += addQuery
		query += addQuery

		idxParam++
		paramData = append(paramData, user_id)
	} else if is_room_joined == "true" {
		if user_id < 1 {
			return &rooms, 0, fmt.Errorf("User ID is required")
		}
		addQuery := " AND rc.id IN (SELECT room_id FROM room_members WHERE user_id = $" + fmt.Sprint(idxParam) + ")"

		total_query += addQuery
		query += addQuery

		idxParam++
		paramData = append(paramData, user_id)
	}

	if search != "" {
		addQuery := " AND (rc.name ILIKE $" + fmt.Sprint(idxParam) +
			" OR rc.code ILIKE $" + fmt.Sprint(idxParam+1) +
			" OR u.username ILIKE $" + fmt.Sprint(idxParam+2) + ")"

		total_query += addQuery
		query += addQuery

		paramData = append(paramData, "%"+search+"%", "%"+search+"%", "%"+search+"%")
		idxParam += 3
	}

	// setup for train rizz exception
	if is_room_train == "true" {
		query += " AND rc.is_train_room = TRUE" //+ fmt.Sprint(idxParam)
		// paramData = append(paramData, "false")
		// idxParam++
	} else {
		query += " AND rc.is_train_room = FALSE" //+ fmt.Sprint(idxParam)
		// paramData = append(paramData, "true")
		// idxParam++
	}

	if filter != "" && filter == "oldest" {
		filterType = "ASC"
	} else {
		filterType = "DESC"
	}

	// total data
	// query total data first bcs we need to know how many data in total and no need to use 2 extra parameter below for offset and per_page
	if err := tx.QueryRow(total_query, paramData...).Scan(&total); err != nil && err != sql.ErrNoRows {
		return &rooms, total, err
	}

	query += " ORDER BY rc.created_at " + filterType + " OFFSET $" + fmt.Sprint(idxParam) + " LIMIT $" + fmt.Sprint(idxParam+1)
	idxParam += 2
	paramData = append(paramData, offset, per_page)

	// all data
	rows, err := tx.Query(query, paramData...)
	if err != nil {
		return &rooms, total, err
	}
	defer rows.Close()

	for rows.Next() {
		var room models.RoomChatDataShow

		if err := rows.Scan(&room.Id, &room.RoomCode, &room.CreatedBy, &room.Username, &room.RoomName, &room.Description, &room.CreatedAt, &room.IsPrivate, &room.IsTrainRoom); err != nil {
			return &rooms, total, err
		}

		rooms = append(rooms, room)
	}

	return &rooms, total, nil
}

func (r *RoomChatRepo) FindByCodeOrAndId(tx *sql.Tx, code string, id int) (*models.RoomChatDataShow, error) {
	var room models.RoomChatDataShow

	if code == "" && id < 1 {
		return &room, fmt.Errorf("Code or/and ID is required")
	}

	query := "SELECT rc.id, rc.code, rc.created_by, u.username, rc.name, rc.description, rc.created_at, rc.is_private, rc.is_train_room, rc.password FROM room_chat rc LEFT JOIN users u ON rc.created_by = u.id WHERE 1=1"

	idx := 1
	paramData := []interface{}{}
	if code != "" {
		query += " AND rc.code = $" + fmt.Sprint(idx)
		idx++
		paramData = append(paramData, code)
	}

	if id > 0 {
		query += " AND rc.id = $" + fmt.Sprint(idx)
		idx++
		paramData = append(paramData, id)
	}

	if err := tx.QueryRow(query, paramData...).Scan(&room.Id, &room.RoomCode, &room.CreatedBy, &room.Username, &room.RoomName, &room.Description, &room.CreatedAt, &room.IsPrivate, &room.IsTrainRoom, &room.Password); err != nil && err != sql.ErrNoRows {
		return &room, err
	}

	return &room, nil
}

func (r *RoomChatRepo) Create(tx *sql.Tx, room *models.RoomChat) error {
	query := "INSERT INTO room_chat (code, created_by, name, description, password, is_private, is_train_room) VALUES ($1, $2, $3, $4, $5, $6, $7)"

	if _, err := tx.Exec(query, room.RoomCode, room.CreatedBy, room.RoomName, room.Description, room.Password, room.IsPrivate, room.IsTrainRoom); err != nil {
		return err
	}

	return nil
}

func (r *RoomChatRepo) Update(tx *sql.Tx, room *models.RoomChat, is_update_password bool) error {

	paramData := []interface{}{room.RoomName, room.Description, room.IsPrivate, room.Id}
	paramCount := len(paramData)

	query := "UPDATE room_chat SET name = $1, description = $2, updated_at = NOW(), password = "

	if is_update_password {
		paramCount++
		paramData = append(paramData, room.Password)

		query += "$" + fmt.Sprint(paramCount)
	} else {
		query += "password"
	}

	query += ", is_private = $3 WHERE id = $4"

	if _, err := tx.Exec(query, paramData...); err != nil {
		return err
	}

	return nil
}

func (r *RoomChatRepo) Delete(tx *sql.Tx, id int) error {
	query := "DELETE FROM room_chat WHERE id = $1"

	if _, err := tx.Exec(query, id); err != nil {
		return err
	}

	return nil
}
