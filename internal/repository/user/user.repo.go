package user

import (
	"database/sql"

	"github.com/momokii/simple-chat-app/internal/models"
)

type UserRepo struct{}

func NewUserRepo() *UserRepo {
	return &UserRepo{}
}

func (r *UserRepo) FindByID(tx *sql.Tx, id int) (*models.User, error) {
	var user models.User

	query := "SELECT id, username, password FROM users WHERE id = $1"

	if err := tx.QueryRow(query, id).Scan(&user.Id, &user.Username, &user.Password); err != nil && err != sql.ErrNoRows {
		return &models.User{}, err
	}

	return &user, nil
}

func (r *UserRepo) FindByUsername(tx *sql.Tx, username string) (*models.User, error) {
	var user models.User

	query := "SELECT id, username, password FROM users WHERE username = $1"

	if err := tx.QueryRow(query, username).Scan(&user.Id, &user.Username, &user.Password); err != nil && err != sql.ErrNoRows {
		return &models.User{}, err
	}

	return &user, nil
}

func (r *UserRepo) Create(tx *sql.Tx, user *models.User) error {
	query := "INSERT INTO users (username, password) VALUES ($1, $2)"

	if _, err := tx.Exec(query, user.Username, user.Password); err != nil {
		return err
	}

	return nil
}

func (r *UserRepo) Update(tx *sql.Tx, user *models.User) error {
	query := "UPDATE users SET username = $1 WHERE id = $2"

	if _, err := tx.Exec(query, user.Username, user.Id); err != nil {
		return err
	}

	return nil
}

func (r *UserRepo) UpdatePassword(tx *sql.Tx, user *models.User) error {
	query := "UPDATE users SET password = $1 WHERE id = $2"

	if _, err := tx.Exec(query, user.Password, user.Id); err != nil {
		return err
	}

	return nil
}
