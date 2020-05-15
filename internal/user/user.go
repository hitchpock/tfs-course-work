package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Storage interface {
	Create(user *User) error
	FindByID(ID int) (*User, error)
	FindByEmail(email string) (*User, error)
	Update(user *User) error
}

// Структура пользователя.
type User struct {
	ID        int
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Birthday  time.Time `json:"birthday,omitempty"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Кастомный Unmarshal пользователя с временем.
func (u *User) UnmarshalJSON(b []byte) error {
	type Alias User

	aux := &struct {
		Birthday string `json:"birthday"`
		*Alias
	}{
		Alias: (*Alias)(u),
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	if aux.Birthday == "" {
		aux.Birthday = "0001-01-01"
	}

	aux.Birthday += "T00:00:00Z"
	t, err := time.Parse(time.RFC3339, aux.Birthday)

	if err != nil {
		return fmt.Errorf("unable to parse time: %s", err)
	}

	u.Birthday = t

	return nil
}

func (u *User) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Birthday  string `json:"birthday,omitempty"`
		Email     string `json:"email"`
	}{
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Birthday:  u.Birthday.Format("2006-01-02"),
		Email:     u.Email,
	})
}

// CreateFromJSON формирует объект пользователя из слайса байт.
func (u *User) CreateFromJSON(body []byte) error {
	err := json.Unmarshal(body, &u)
	if err != nil {
		return fmt.Errorf("invalid input: %s", err)
	}

	if u.FirstName == "" || u.LastName == "" || u.Email == "" || u.Password == "" {
		return errors.New("invalid input")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("func bcrypt.GenerateFromPassword is crached: %s", err)
	}

	u.Password = string(passwordHash)

	return nil
}

// Обновляет структуру пользователя
func (u *User) Update(desUser *User) {
	u.FirstName = desUser.FirstName
	u.LastName = desUser.LastName
	u.Email = desUser.Email
	u.Password = desUser.Password
	u.UpdatedAt = time.Now()

	if (desUser.Birthday != time.Time{}) {
		u.Birthday = desUser.Birthday
	}
}
