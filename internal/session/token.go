package session

import (
	"encoding/base32"
	"encoding/json"
	"fmt"
)

// Структура токена
type BearerToken struct {
	Token string `json:"bearer"`
}

// Создание токена сессии
func CreateToken(session *Session) string {
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		fmt.Printf("session.Marshal is crached: %s", err)
	}

	token := base32.StdEncoding.EncodeToString(sessionJSON)

	return token
}

// Декодирование токена сессии в сессию
func DecodeToken(token string) (*Session, error) {
	sesJSON, err := base32.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("func base32.StdEncoding.DecodeString is crashed: %s", err)
	}

	var ses Session

	err = json.Unmarshal(sesJSON, &ses)
	if err != nil {
		return nil, fmt.Errorf("func json.Unmarshal is crashed: %s", err)
	}

	return &ses, nil
}
