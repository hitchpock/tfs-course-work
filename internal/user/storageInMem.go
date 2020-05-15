package user

import (
	"fmt"
	"time"
)

// Структура хранилища в памяти.
type StorageInMemory struct {
	storage map[string]User
}

func CreateStorageInMemory() *StorageInMemory {
	return &StorageInMemory{storage: make(map[string]User)}
}

// Добавление пользователя в хранилище.
func (s *StorageInMemory) Create(u *User) error {
	if _, ok := s.storage[u.Email]; ok {
		return fmt.Errorf("user %s is already registered", u.Email)
	}

	u.ID = len(s.storage)
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()

	s.storage[u.Email] = *u

	return nil
}

func (s *StorageInMemory) FindByEmail(email string) (*User, error) {
	user, ok := s.storage[email]
	if !ok {
		return nil, fmt.Errorf("user %s not found", email)
	}

	return &user, nil
}

func (s *StorageInMemory) FindByID(id int) (*User, error) {
	users := s.storage
	for _, user := range users {
		if user.ID == id {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("user %d not found", id)
}

func (s *StorageInMemory) Update(user *User) error {
	s.storage[user.Email] = *user
	return nil
}
