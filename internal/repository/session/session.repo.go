package session

import (
	"database/sql"

	"github.com/momokii/simple-chat-app/internal/models"
)

type SessionRepo struct{}

func NewSessionRepo() *SessionRepo {
	return &SessionRepo{}
}

func (r *SessionRepo) FindSession(tx *sql.Tx, id_session string, id_user int) (*models.Session, error) {
	var session models.Session

	query := "SELECT id, user_id, session_id, expires_at FROM sessions WHERE session_id = $1 AND user_id = $2"

	if err := tx.QueryRow(query, id_session, id_user).Scan(&session.Id, &session.UserId, &session.SessionId, &session.ExpiresAt); err != nil && err != sql.ErrNoRows {
		return &session, err
	}
	return &session, nil
}

func (r *SessionRepo) Create(tx *sql.Tx, session *models.SessionCreate) error {
	query := "INSERT INTO sessions (user_id, session_id, expires_at, created_at) VALUES ($1, $2, $3, $4)"

	if _, err := tx.Exec(query, session.UserId, session.SessionId, session.ExpiresAt, session.CreatedAt); err != nil {
		return err
	}

	return nil
}

func (r *SessionRepo) Delete(tx *sql.Tx, id_session string, id_user int) error {
	query := "DELETE FROM sessions WHERE session_id = $1 AND user_id = $2"

	if _, err := tx.Exec(query, id_session, id_user); err != nil {
		return err
	}

	return nil
}

func (r *SessionRepo) DeleteExpiredSession(tx *sql.Tx, time_now string) error {
	query := "DELETE FROM sessions WHERE expires_at < $1"

	if _, err := tx.Exec(query, time_now); err != nil {
		return err
	}

	return nil
}
