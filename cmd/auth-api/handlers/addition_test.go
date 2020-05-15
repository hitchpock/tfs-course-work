package handlers

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildErrorJSON(t *testing.T) {
	assert := assert.New(t)
	errError := errors.New("error type Error")
	errString := "error type string"

	str, _ := buildFmtError(errError.Error())
	expected := `{"error":"error type Error"}`
	assert.Equal([]byte(expected), str, "buildErrorJSON(%q) = %s, want %s", errError, string(str), expected)

	str, _ = buildFmtError(errString)
	expected = `{"error":"error type string"}`
	assert.Equal([]byte(expected), str, "buildErrorJSON(%q) = %s, want %s", errString, string(str), expected)
}

func TestGetUserFromBody(t *testing.T) {
	assert := assert.New(t)

	_, err := getUserFromBody(bytes.NewBuffer([]byte("")))
	assert.NotNil(err)

	userWithBirthday := []byte(`{"first_name":"Ivan","last_name":"Ivanov","birthday":"1980-01-02","email":"example@example.com","password":"1234"}`)
	_, err = getUserFromBody(bytes.NewBuffer(userWithBirthday))
	assert.Nil(err)

	userWithoutBirthday := []byte(`{"first_name":"Ivan","last_name":"Ivanov","email":"example@example.com","password":"1234"}`)
	_, err = getUserFromBody(bytes.NewBuffer(userWithoutBirthday))
	assert.Nil(err)

	userWithoutField := []byte(`{"first_name":"Ivan","email":"example@example.com","password":"1234"}`)
	_, err = getUserFromBody(bytes.NewBuffer(userWithoutField))
	assert.NotNil(err)
}

func TestValidQuery(t *testing.T) {
	type testCase struct {
		Name      string
		In        []byte
		Subslices [][]byte
		Expected  bool
	}

	assert := assert.New(t)

	testCases := []testCase{
		{Name: "valid query", In: []byte(`{\t"email": example,\n\t"password": 1234}`),
			Subslices: [][]byte{[]byte("email"), []byte("password")}, Expected: true},
		{Name: "invalid query", In: []byte(`{\t"emal": example,\n\t"password": 1234}`),
			Subslices: [][]byte{[]byte("email"), []byte("password")}, Expected: false},
		{Name: "empty query", In: []byte(""),
			Subslices: [][]byte{[]byte("email"), []byte("password")}, Expected: false},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.Name, func(t *testing.T) {
			actual := validQuery(tc.In, tc.Subslices...)

			assert.Equal(tc.Expected, actual, "validQuery(%q, %v) = %v, want = %v", tc.In, tc.Subslices, actual, tc.Expected)
		})
	}
}

func TestGetSignInDataFromBody(t *testing.T) {
	type testCase struct {
		Name         string
		In           []byte
		ExpectedUser *SignInData
	}

	assert := assert.New(t)

	testCases := []testCase{
		{Name: "invalid body", In: []byte(`{"email":"test@test.com","password":"1234"}`),
			ExpectedUser: &SignInData{Email: "test@test.com", Password: "1234"}},
		{Name: "invalid email field", In: []byte(`{"emil":"test@test.com","password":"1234"}`),
			ExpectedUser: &SignInData{Email: "", Password: "1234"}},
		{Name: "invalid form", In: []byte(``),
			ExpectedUser: nil},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.Name, func(t *testing.T) {
			actual, _ := getSignInDataFromBody(bytes.NewBuffer(tc.In))

			assert.Equal(tc.ExpectedUser, actual, "getSignInDataFromBody(%q) = %v, want = %v", tc.In, actual, tc.ExpectedUser)
		})
	}
}

func TestErrorJSON(t *testing.T) {
	type testCase struct {
		Name     string
		In       string
		Expected string
	}

	assert := assert.New(t)

	testCases := []testCase{
		{Name: "correct", In: "error", Expected: "{\n\t\"error\": \"error\"\n}"},
		{Name: "empty", In: "", Expected: "{\n\t\"error\": \"\"\n}"},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.Name, func(t *testing.T) {
			actual := errorJSON(tc.In)

			assert.Equal(tc.Expected, actual, "errorJSON(%q) = %q, want %q", tc.In, actual, tc.Expected)
		})
	}
}
