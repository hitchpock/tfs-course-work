package session

import (
	"fmt"
)

// Структура хранилища сессий in-memory.
type StorageInMemory struct {
	storage []Session
}

// CreateStorageInMemory возвращает указатель на хранилище in-memory.
func CreateStorageInMemory() *StorageInMemory {
	return &StorageInMemory{storage: []Session{}}
}

// addSessionInStorage добавляет сессию в хранилище in-memory.
func (s *StorageInMemory) Create(ses *Session) error {
	s.storage = append(s.storage, *ses)

	return nil
}

func (s *StorageInMemory) FindByUserID(id int) (*Session, error) {
	for count := len(s.storage) - 1; count >= 0; count-- {
		ses := s.storage[count]
		if ses.UserID == id {
			return &ses, nil
		}
	}

	return nil, fmt.Errorf("ID %d not found", id)
}

// func (s *StorageInMemory) GetSessions() []Session {
// 	var sessions []Session

// 	for _, row := range s.storage {
// 		s := row
// 		sessions = append(sessions, s)
// 	}

// 	return sessions
// }
