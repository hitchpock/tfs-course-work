package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"gitlab.com/hitchpock/tfs-course-work/internal/robot"
	"gitlab.com/hitchpock/tfs-course-work/internal/user"
	"gitlab.com/hitchpock/tfs-course-work/web"

	"golang.org/x/crypto/bcrypt"
)

// SignInData структура аунтификации пользователя.
type SignInData struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func errorJSON(stringErr string) string {
	return fmt.Sprintf("{\n\t\"error\": \"%s\"\n}", stringErr)
}

// CreatePasswordHash создает хэш пароля пользователя.
func CreatePasswordHash(password string) (string, error) {
	pwdHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(pwdHash), err
}

// CheckPasswordHash сравнивает хэш с паролем.
func CheckPasswordHash(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// getUserFromBody считывает данные из запроса и возвращает указатель на объект пользователя из тела запроса.
func getUserFromBody(body io.Reader) (*user.User, error) {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("func ioutil.ReadAll is crashed: %s", err)
	}

	var u user.User
	if err := u.CreateFromJSON(b); err != nil {
		return nil, fmt.Errorf("func CreateFromJSON is crashed: %s", err)
	}

	return &u, nil
}

// getSignInData считывает данные из запроса возвращает указатель на объект для аунтификации.
func getSignInDataFromBody(body io.Reader) (*SignInData, error) {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("func ioutil.ReadAll is crashed: %s", err)
	}

	var u SignInData
	if err = json.Unmarshal(b, &u); err != nil {
		return nil, fmt.Errorf("invalid input: %s", err)
	}

	return &u, nil
}

// getRobotFromBody считывает данные из запроса и возвращает указатель на робота.
func getRobotFromBody(body io.Reader) (*robot.Robot, error) {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("func ioutil.ReadAll is crashed: %s", err)
	}

	if !validQuery(b, []byte("owner_user_id"), []byte("is_favourite"), []byte("is_active")) { //nolint:misspell
		return nil, errors.New("ivalid input")
	}

	var r robot.Robot
	if err = json.Unmarshal(b, &r); err != nil || r.OwnerUserID == 0 {
		return nil, fmt.Errorf("invalid input: %s", err)
	}

	return &r, nil
}

func validQuery(body []byte, subslices ...[]byte) bool {
	for _, subslice := range subslices {
		if !bytes.Contains(body, subslice) {
			return false
		}
	}

	return true
}

func renderTemplate(w http.ResponseWriter, name string, template string, viewModel interface{}) {
	tmpl, ok := web.Templates[name]
	if !ok {
		sendError(w, "can't find template", http.StatusInternalServerError)
	}

	err := tmpl.ExecuteTemplate(w, template, viewModel)
	if err != nil {
		sendError(w, "error on server", http.StatusInternalServerError)
	}
}

func sendError(w http.ResponseWriter, error string, code int) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	fmt.Fprintln(w, errorJSON(error))
}
