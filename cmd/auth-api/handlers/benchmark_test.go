package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
)

// ErrorJSON структура возвращаемой ошибки.
type FmtError struct {
	Error string `json:"error"`
}

// buildErrorJSON создает ошибку в формате JSON.
func buildFmtError(stringError string) ([]byte, error) {
	e := FmtError{Error: stringError}

	stringJSON, err := json.Marshal(e)
	if err != nil {
		return []byte(""), fmt.Errorf("func Marshal is crashed: %s", err)
	}

	return stringJSON, nil
}

func BenchmarkMarshal(b *testing.B) {
	err := "error on server"

	for i := 0; i < b.N; i++ {
		_, _ = buildFmtError(err)
	}
}

func BenchmarkSprinf(b *testing.B) {
	err := "error on server"

	for i := 0; i < b.N; i++ {
		_ = errorJSON(err)
	}
}

func BenchmarkStrconvParseInt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = strconv.ParseInt("1234", 10, 64)
	}
}

func BenchmarkStrconvAtoi(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = strconv.Atoi("1234")
	}
}
