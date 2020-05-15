package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"gitlab.com/hitchpock/tfs-course-work/internal/session"
	"gitlab.com/hitchpock/tfs-course-work/internal/user"
	"gitlab.com/hitchpock/tfs-course-work/pkg/log"
)

const (
	urlSignUp = "/api/v1/signup"
	urlSignIn = "/api/v1/signin"

	correctSignUp     = `{"first_name":"Ivan","last_name":"Ivanov","birthday":"1980-01-02","email":"e@example.com","password":"1234"}`
	correctSignIn     = `{"email":"e@example.com","password":"1234"}`
	randomString      = `dcmldskcmsdc`
	invalidUpdateUser = `{"first_name":"Ivan","birthday":"1980-01-02","email":"e@example.com","password":"1234"}`
	validUpdateUser   = `{"first_name":"Victor","last_name":"Ivanov","birthday":"1990-01-02","email":"e@example.com","password":"1234"}`
)

func TestSingUp(t *testing.T) {
	type testCase struct {
		Name         string
		In           []byte
		ExpectedCode int
	}

	assert := assert.New(t)
	logger := log.NewSugarLogger()
	sessionStorage := session.CreateStorageInMemory()

	testCases := []testCase{
		{Name: "Empty body", In: []byte(""), ExpectedCode: http.StatusBadRequest},
		{Name: "User without birthday", In: []byte(`{"first_name":"Ivan","last_name":"Ivanov","email":"e@example.com","password":"1234"}`), ExpectedCode: http.StatusCreated},
		{Name: "User withoutrequired fields", In: []byte(`{"first_name":"Ivan","email":"e@example.com","password":"1234"}`), ExpectedCode: http.StatusBadRequest},
		{Name: "User with birthday", In: []byte(`{"first_name":"Ivan","last_name":"Ivanov","birthday":"1980-01-02","email":"e@example.com","password":"1234"}`), ExpectedCode: http.StatusCreated},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			userStorage := user.CreateStorageInMemory()
			h := NewHandler(logger, sessionStorage, userStorage, nil, nil)
			handler := http.HandlerFunc(h.SignUp)
			request := httptest.NewRequest(http.MethodPost, urlSignUp, bytes.NewBuffer(tc.In))
			recoder := httptest.NewRecorder()
			handler.ServeHTTP(recoder, request)
			respBody := recoder.Body.String()

			if recoder.Code != tc.ExpectedCode {
				assert.Equal(tc.ExpectedCode, recoder.Code, "Wrong http code, response: %s, request: %q", respBody, tc.In)
			}
		})
	}

	repitSignUp := testCase{Name: "Repited SignUp", In: []byte(`{"first_name":"Ivan","last_name":"Ivanov","birthday":"1980-01-02","email":"e@example.com","password":"1234"}`), ExpectedCode: http.StatusConflict}
	userStorage := user.CreateStorageInMemory()
	h := NewHandler(logger, sessionStorage, userStorage, nil, nil)
	handler := http.HandlerFunc(h.SignUp)

	for i := 0; i < 2; i++ {
		i := i

		t.Run(repitSignUp.Name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, urlSignUp, bytes.NewBuffer(repitSignUp.In))
			recoder := httptest.NewRecorder()
			handler.ServeHTTP(recoder, request)

			if i == 1 {
				respBody := recoder.Body.String()

				if recoder.Code != repitSignUp.ExpectedCode {
					assert.Equal(repitSignUp.ExpectedCode, recoder.Code, "Wrong http code, response: %s, request: %q", respBody, repitSignUp.In)
				}
			}
		})
	}
}

func TestSignIn(t *testing.T) {
	type testCase struct {
		Name         string
		In           []byte
		ExpectedCode int
	}

	assert := assert.New(t)
	logger := log.NewSugarLogger()
	sessionStorage := session.CreateStorageInMemory()
	userStorage := user.CreateStorageInMemory()
	h := NewHandler(logger, sessionStorage, userStorage, nil, nil)

	testCases := []testCase{
		{Name: "Empty body", In: []byte(""), ExpectedCode: http.StatusBadRequest},
		{Name: "Invalid input", In: []byte("klsdcm"), ExpectedCode: http.StatusBadRequest},
		{Name: "Invalid input", In: []byte(`{"mail":"e@example.com","pwd":"1234"}`), ExpectedCode: http.StatusBadRequest},
		{Name: "Non-exist user", In: []byte(`{"email":"ex@example.com","password":"1234"}`), ExpectedCode: http.StatusBadRequest},
		{Name: "Invalid password", In: []byte(`{"email":"e@example.com","password":"1233"}`), ExpectedCode: http.StatusBadRequest},
		{Name: "Correct SignIn", In: []byte(`{"email":"e@example.com","password":"1234"}`), ExpectedCode: http.StatusOK},
	}

	setupSignUp(h, t)
	handler := http.HandlerFunc(h.SignIn)

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, urlSignIn, bytes.NewBuffer(tc.In))
			recoder := httptest.NewRecorder()
			handler.ServeHTTP(recoder, request)
			respBody := recoder.Body.String()

			if recoder.Code != tc.ExpectedCode {
				assert.Equal(tc.ExpectedCode, recoder.Code, "Wrong http code, response: %s, request: %q", respBody, tc.In)
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	type testCase struct {
		Name         string
		Path         string
		Body         string
		ExpectedCode int
	}

	assert := assert.New(t)
	logger := log.NewSugarLogger()
	sessionStorage := session.CreateStorageInMemory()
	userStorage := user.CreateStorageInMemory()
	h := NewHandler(logger, sessionStorage, userStorage, nil, nil)

	testCases := []testCase{
		{Name: "Someone else id", Path: "/api/v1/users/12", Body: validUpdateUser, ExpectedCode: http.StatusForbidden},
		{Name: "Random body", Path: "/api/v1/users/0", Body: randomString, ExpectedCode: http.StatusBadRequest},
		{Name: "Invalid body", Path: "/api/v1/users/0", Body: invalidUpdateUser, ExpectedCode: http.StatusBadRequest},
	}

	r := chi.NewRouter()
	r.With(h.getParamID).With(h.authentication, h.authorization).Put("/api/v1/users/{id}", h.UpdateUser)

	ts := httptest.NewServer(r)
	defer ts.Close()

	setupSignUp(h, t)
	tokenBytes := setupSignIn(h, t)

	var token session.BearerToken
	err := json.Unmarshal(tokenBytes, &token)
	assert.NoError(err)

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			auth := fmt.Sprintf("Bearer %s", token.Token)
			recoder, code := testRequestWithAuth(t, ts, http.MethodPut, tc.Path, auth, bytes.NewBuffer([]byte(tc.Body)))
			defer recoder.Body.Close()

			if code != tc.ExpectedCode {
				assert.Equal(tc.ExpectedCode, code, "Wrong http code, response: %s, request: %q", recoder.Body, tc.Body)
			}
		})
	}
}

// setupSignUp регистрирует пользователя
func setupSignUp(h *Handler, t *testing.T) {
	assert := assert.New(t)
	handler := http.HandlerFunc(h.SignUp)
	req := httptest.NewRequest(http.MethodPost, urlSignUp, bytes.NewBuffer([]byte(correctSignUp)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(http.StatusCreated, rec.Code)
	t.Logf("signup for %s is OK", correctSignUp)
}

// setupSignIn создает новую сессию и возвращает токен аунтификации
func setupSignIn(h *Handler, t *testing.T) []byte {
	assert := assert.New(t)
	handler := http.HandlerFunc(h.SignIn)
	req := httptest.NewRequest(http.MethodPost, urlSignIn, bytes.NewBuffer([]byte(correctSignIn)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(http.StatusOK, rec.Code)
	t.Logf("signin for %s is OK", correctSignIn)

	return rec.Body.Bytes()
}

// testRequestWithAuth отправляет запрос на тестовый сервер и возвращает ответ
func testRequestWithAuth(t *testing.T, ts *httptest.Server, method, path string, header string, body io.Reader) (*http.Response, int) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
		return nil, 0
	}

	req.Header.Set("Authorization", header)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
		return nil, 0
	}

	return resp, resp.StatusCode
}
