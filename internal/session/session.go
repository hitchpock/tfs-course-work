package session

import (
	"time"
)

const (
	tokenValidTime = 30 * time.Minute
)

// Интерфейс сессии
type Storage interface {
	Create(ses *Session) error
	FindByUserID(ID int) (*Session, error)
}

// Структура сессии
type Session struct {
	SessionID  string    `json:"session_id"`
	UserID     int       `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
	ValidUntil time.Time `json:"valid_until"`
}

func (s *Session) Equal(session *Session) bool {
	if s.SessionID != session.SessionID || s.UserID != session.UserID {
		return false
	}

	if s.CreatedAt.Unix() != session.CreatedAt.Unix() || s.ValidUntil.Unix() != session.ValidUntil.Unix() {
		return false
	}

	return true
}

func NewSession(userID int) *Session {
	session := &Session{UserID: userID, CreatedAt: time.Now(), ValidUntil: time.Now().Add(tokenValidTime)}
	token := CreateToken(session)
	session.SessionID = token

	return session
}
