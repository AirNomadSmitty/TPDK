package mappers

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	UserID       int64
	Username     string
	PasswordHash string
}

func (user *User) ValidatePassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return false
	}
	return true
}

type UserMapper struct {
	db *sql.DB
}

func MakeUserMapper(db *sql.DB) *UserMapper {
	return &UserMapper{db}
}

func (mapper *UserMapper) GetFromUsername(username string) (*User, error) {
	user := &User{}

	err := mapper.db.QueryRow("SELECT user_id, username, password_hash FROM users WHERE username=?", username).Scan(&user.UserID, &user.Username, &user.PasswordHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (mapper *UserMapper) GetFromUserID(userID int64) (*User, error) {
	user := &User{}
	err := mapper.db.QueryRow("SELECT user_id, username, password_hash FROM users WHERE user_id=?", userID).Scan(&user.UserID, &user.Username, &user.PasswordHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return user, nil

}

func (mapper *UserMapper) Update(user *User) error {
	if user.UserID != 0 {
		_, err := mapper.db.Exec("UPDATE users SET username = ?, password_hash = ?", user.Username, user.PasswordHash)
		return err
	}
	return errors.New("Can only update existing users")
}

func (mapper *UserMapper) Create(username string, passwordHash string) (*User, error) {
	result, err := mapper.db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, passwordHash)
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	user := &User{id, username, passwordHash}
	return user, nil
}
