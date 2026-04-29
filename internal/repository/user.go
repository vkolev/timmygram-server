package repository

import (
	"database/sql"
	"errors"
	"timmygram/internal/model"
)

type UserRepository interface {
	Create(user *model.User) error
	FindByUsername(username string) (*model.User, error)
	FindByID(id int) (*model.User, error)
	HasUsers() (bool, error)
}

type SQLiteUserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &SQLiteUserRepository{db: db}
}

func (r *SQLiteUserRepository) Create(user *model.User) error {
	_, err := r.db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", user.Username, user.Password)
	return err
}

func (r *SQLiteUserRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	row := r.db.QueryRow("SELECT id, username, password FROM users WHERE username = ?", username)
	err := row.Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *SQLiteUserRepository) FindByID(id int) (*model.User, error) {
	var user model.User
	row := r.db.QueryRow("SELECT id, username, password FROM users WHERE id = ?", id)
	err := row.Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *SQLiteUserRepository) HasUsers() (bool, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count > 0, err
}
