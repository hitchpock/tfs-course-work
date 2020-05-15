package postgres

import (
	"database/sql"
	"fmt"

	"gitlab.com/hitchpock/tfs-course-work/internal/session"
)

var _ session.Storage = &SessionStorage{}

type SessionStorage struct {
	statementStorage

	createStmt   *sql.Stmt
	findByUserID *sql.Stmt
}

func NewSessionStorage(db *DB) (*SessionStorage, error) {
	s := &SessionStorage{statementStorage: newStatementStorage(db)}

	stmts := []stmt{
		{Query: createSessionQuery, Dst: &s.createStmt},
		{Query: findByUserIDQuery, Dst: &s.findByUserID},
	}

	if err := s.initStatements(stmts); err != nil {
		return nil, fmt.Errorf("can't init statements: %s", err)
	}

	return s, nil
}

const sessionFields = `session_id, user_id, created_at, valid_until`

func scanSession(scanner sqlScanner, s *session.Session) error {
	return scanner.Scan(&s.SessionID, &s.UserID, &s.CreatedAt, &s.ValidUntil)
}

const createSessionQuery = `INSERT INTO sessions(` + sessionFields + `) ` +
	`VALUES ($1, $2, $3, $4)`

func (s *SessionStorage) Create(ses *session.Session) error {
	tx, err := s.db.Session.Begin()
	if err != nil {
		return fmt.Errorf("unable to start a transaction: %s", err)
	}

	if _, err = tx.Stmt(s.createStmt).Exec(ses.SessionID, ses.UserID, ses.CreatedAt, ses.ValidUntil); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("can't insert session in database: %s", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("can't commit in sessionStorage: %s", err)
	}

	return nil
}

const findByUserIDQuery = `SELECT ` + sessionFields + ` FROM sessions WHERE user_id = $1 ORDER BY created_at DESC`

func (s *SessionStorage) FindByUserID(id int) (*session.Session, error) {
	var ses session.Session

	row := s.findByUserID.QueryRow(id)
	if err := scanSession(row, &ses); err != nil {
		return nil, fmt.Errorf("can't scan session: %s", err)
	}

	return &ses, nil
}
