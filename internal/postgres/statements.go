package postgres

import (
	"database/sql"
	"fmt"
)

type statementStorage struct {
	db         *DB
	statements []*sql.Stmt
}

func newStatementStorage(db *DB) statementStorage {
	return statementStorage{db: db}
}

func (s *statementStorage) Close() error {
	for _, stmt := range s.statements {
		if err := stmt.Close(); err != nil {
			return fmt.Errorf("can't close statement: %s", err)
		}
	}

	return nil
}

type stmt struct {
	Query string
	Dst   **sql.Stmt
}

func (s *statementStorage) initStatements(statements []stmt) error {
	for i := range statements {
		statement, err := s.db.Session.Prepare(statements[i].Query)
		if err != nil {
			return fmt.Errorf("can't prepare query %q: %s", statements[i].Query, err)
		}

		*statements[i].Dst = statement
		s.statements = append(s.statements, statement)
	}

	return nil
}
