package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"gitlab.com/hitchpock/tfs-course-work/internal/user"
)

var _ user.Storage = &UserStorage{}

type UserStorage struct {
	statementStorage

	createStmt      *sql.Stmt
	findByIDStmt    *sql.Stmt
	findByEmailStmt *sql.Stmt
	updateStmt      *sql.Stmt
}

func NewUserStorage(db *DB) (*UserStorage, error) {
	s := &UserStorage{statementStorage: newStatementStorage(db)}

	stmts := []stmt{
		{Query: createUserQuery, Dst: &s.createStmt},
		{Query: findUserByIDQuery, Dst: &s.findByIDStmt},
		{Query: findUserByEmailQuery, Dst: &s.findByEmailStmt},
		{Query: updateUserQuery, Dst: &s.updateStmt},
	}

	if err := s.initStatements(stmts); err != nil {
		return nil, fmt.Errorf("can't init statements: %s", err)
	}

	return s, nil
}

const userFields = `id, first_name, last_name, birthday, email, password, ` +
	`created_at, updated_at`

func scanUser(scanner sqlScanner, u *user.User) error {
	return scanner.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Birthday, &u.Email, &u.Password,
		&u.CreatedAt, &u.UpdatedAt)
}

const createUserQuery = `INSERT INTO users(first_name, last_name, birthday, email, password, created_at, updated_at) ` +
	`VALUES ($1, $2, $3, $4, $5, $6, $7)`

func (s *UserStorage) Create(u *user.User) error {
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()

	tx, err := s.db.Session.Begin()
	if err != nil {
		return fmt.Errorf("unable to start a transaction: %s", err)
	}

	if _, err = tx.Stmt(s.createStmt).Exec(u.FirstName, u.LastName, u.Birthday, u.Email, u.Password, u.CreatedAt, u.UpdatedAt); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("user %s is already registered", u.Email)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("can't commit in userStorage: %s", err)
	}

	return nil
}

const findUserByIDQuery = `SELECT ` + userFields + ` FROM users WHERE id = $1`

func (s *UserStorage) FindByID(id int) (*user.User, error) {
	var u user.User

	row := s.findByIDStmt.QueryRow(id)
	if err := scanUser(row, &u); err != nil {
		return nil, fmt.Errorf("can't scan user: %s", err)
	}

	return &u, nil
}

const findUserByEmailQuery = `SELECT ` + userFields + ` FROM users WHERE email = $1`

func (s *UserStorage) FindByEmail(email string) (*user.User, error) {
	var u user.User

	row := s.findByEmailStmt.QueryRow(email)
	if err := scanUser(row, &u); err != nil {
		return nil, fmt.Errorf("can't scan user: %s", err)
	}

	return &u, nil
}

const updateUserQuery = `UPDATE users ` +
	`SET first_name = $1, last_name = $2, birthday = $3, email = $4, password = $5, updated_at = $6 ` +
	`WHERE id = $7`

func (s *UserStorage) Update(u *user.User) error {
	u.UpdatedAt = time.Now()

	tx, err := s.db.Session.Begin()
	if err != nil {
		return fmt.Errorf("unable to start a transaction: %s", err)
	}

	if _, err = tx.Stmt(s.updateStmt).Exec(u.FirstName, u.LastName, u.Birthday, u.Email, u.Password, u.UpdatedAt, u.ID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("can't update user: %s", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("can't commit in userStorage: %s", err)
	}

	return nil
}
