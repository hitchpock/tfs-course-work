package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"gitlab.com/hitchpock/tfs-course-work/internal/session"
	"gitlab.com/hitchpock/tfs-course-work/internal/user"
	"gitlab.com/hitchpock/tfs-course-work/pkg/log"
)

const (
	urlUpdateUser = "/api/v1/users/0"
)

func TestGetValidSession(t *testing.T) {
	type testCase struct {
		Name            string
		Header          string
		ExpectedCode    int
		ExpectedMessage string
	}

	assert, logger, sessionStorage, userStorage := prepare(t)
	h := NewHandler(logger, sessionStorage, userStorage, nil, nil)
	ctx := context.WithValue(context.Background(), idKey{}, 1)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK\n")); err != nil {
			t.Logf("unable to write response: %s", err)
		}
	})

	handler := h.authentication(nextHandler)
	fakeStorage := session.CreateStorageInMemory()
	fakeSession := session.NewSession(1)
	_ = fakeStorage.Create(fakeSession)
	fakeTokenOne := fmt.Sprintf("Bearer %s", fakeSession.SessionID)
	_ = sessionStorage.Create(session.NewSession(2))
	fakeSession = &session.Session{UserID: 2, CreatedAt: time.Now().Add(5 * time.Minute), ValidUntil: time.Now().Add(30 * time.Minute)}
	token := session.CreateToken(fakeSession)
	fakeTokenTwo := fmt.Sprintf("Bearer %s", token)

	validSession := session.NewSession(3)
	_ = sessionStorage.Create(validSession)
	token = validSession.SessionID
	validToken := fmt.Sprintf("Bearer %s", token)

	testCases := []testCase{
		{Name: "Empty header", Header: "", ExpectedCode: http.StatusBadRequest, ExpectedMessage: errorJSON("scheme not found")},
		{Name: "Empty token v1", Header: "Bearer", ExpectedCode: http.StatusBadRequest, ExpectedMessage: errorJSON("scheme not found")},
		{Name: "Empty token v2", Header: "Bearer ", ExpectedCode: http.StatusUnauthorized, ExpectedMessage: errorJSON("token not found")},
		{Name: "Not supported scheme", Header: "Base 1234", ExpectedCode: http.StatusBadRequest, ExpectedMessage: errorJSON("the authentication scheme is not supported")},
		{Name: "Invalid token", Header: "Bearer 11111", ExpectedCode: http.StatusBadRequest, ExpectedMessage: errorJSON("invalid token")},
		{Name: "Token from other storage", Header: fakeTokenOne, ExpectedCode: http.StatusNotFound, ExpectedMessage: errorJSON("session not found")},
		{Name: "Token with fake time", Header: fakeTokenTwo, ExpectedCode: http.StatusUnauthorized, ExpectedMessage: errorJSON("invalid token")},
		{Name: "Valid token", Header: validToken, ExpectedCode: http.StatusOK, ExpectedMessage: "OK"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPut, urlUpdateUser, nil)
			request.Header.Set("Authorization", tc.Header)
			recoder := httptest.NewRecorder()
			handler.ServeHTTP(recoder, request.WithContext(ctx))
			respBody := recoder.Body.String()

			if recoder.Code != tc.ExpectedCode {
				assert.Equal(tc.ExpectedCode, recoder.Code, "Wrong http code, response: %s, request: %q", respBody, tc.Header)
			}
			assert.Equal(tc.ExpectedMessage+"\n", respBody)
		})
	}
}

func TestGetParmID(t *testing.T) {
	type testCase struct {
		Name         string
		In           string
		ExpectedCode int
	}

	assert, logger, sessionStorage, userStorage := prepare(t)
	h := NewHandler(logger, sessionStorage, userStorage, nil, nil)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := chi.NewRouter()
	r.With(h.getParamID).Put("/api/v1/users/{id}", nextHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	testCases := []testCase{
		{Name: "Empty id", In: "/api/v1/users/", ExpectedCode: http.StatusNotFound},
		{Name: "Chars", In: "/api/v1/users/@123", ExpectedCode: http.StatusBadRequest},
		{Name: "Long id", In: "/api/v1/users/11111111111111111111", ExpectedCode: http.StatusBadRequest},
		{Name: "Negative id", In: "/api/v1/users/-18", ExpectedCode: http.StatusBadRequest},
		{Name: "Valid id", In: "/api/v1/users/1", ExpectedCode: http.StatusOK},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			recoder, code := testRequest(t, ts, "PUT", tc.In, nil)
			defer recoder.Body.Close()

			if code != tc.ExpectedCode {
				assert.Equal(tc.ExpectedCode, code, "Wrong http code, response: %s, request: %q", recoder.Body, tc.In)
			}
		})
	}
}

func prepare(t *testing.T) (*assert.Assertions, *log.SugarLogger, *session.StorageInMemory, *user.StorageInMemory) {
	assert := assert.New(t)
	logger := log.NewSugarLogger()
	sessionStorage := session.CreateStorageInMemory()
	userStorage := user.CreateStorageInMemory()

	return assert, logger, sessionStorage, userStorage
}

func testRequest(t *testing.T, ts *httptest.Server, method, path string, body io.Reader) (*http.Response, int) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
		return nil, 0
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
		return nil, 0
	}

	return resp, resp.StatusCode
}
