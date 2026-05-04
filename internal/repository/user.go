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
	List() ([]*model.User, error)
	UpdatePassword(userID int, hashedPassword string) error
	Delete(id int) error
}

type SQLiteUserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &SQLiteUserRepository{db: db}
}

func (r *SQLiteUserRepository) Create(user *model.User) error {
	_, err := r.db.Exec(
		"INSERT INTO users (username, password, is_owner) VALUES (?, ?, ?)",
		user.Username, user.Password, user.IsOwner,
	)
	return err
}

func (r *SQLiteUserRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	row := r.db.QueryRow("SELECT id, username, password, is_owner FROM users WHERE username = ?", username)
	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.IsOwner)
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
	row := r.db.QueryRow("SELECT id, username, password, is_owner FROM users WHERE id = ?", id)
	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.IsOwner)
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

func (r *SQLiteUserRepository) List() ([]*model.User, error) {
	rows, err := r.db.Query("SELECT id, username, is_owner FROM users ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.IsOwner); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

func (r *SQLiteUserRepository) UpdatePassword(userID int, hashedPassword string) error {
	_, err := r.db.Exec("UPDATE users SET password = ? WHERE id = ?", hashedPassword, userID)
	return err
}

func (r *SQLiteUserRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}
