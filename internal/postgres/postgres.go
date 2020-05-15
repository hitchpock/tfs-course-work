package postgres

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // init posgres driver
	"gitlab.com/hitchpock/tfs-course-work/pkg/log"
)

// Структура базы данных.
type DB struct {
	Session *sql.DB
	Logger  log.Logger
}

// Структура конфигурации базы данных.
type Config struct {
	URL             string
	ConnMaxLifetime time.Duration
	MaxIdleConns    int
	MaxOpenConns    int
}

// New возвращает указатель на новую структуру базы данных.
func New(logger log.Logger, cfg Config) (*DB, error) {
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("can't open connection to postgres: %s", err)
	}

	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetMaxOpenConns(cfg.MaxOpenConns)

	return &DB{Session: db, Logger: logger}, nil
}

// CheckConnection проверяет соединение с базой данных.
func (d *DB) CheckConnection() (err error) {
	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err = d.Session.Ping(); err == nil {
			d.Logger.Info("connection with database is established")
			return
		}

		attemptPause := time.Duration(attempt) * time.Second
		d.Logger.Warnf("ping %d to database return with error: %s", attempt, err)
		time.Sleep(attemptPause)
	}

	return fmt.Errorf("connection with database is not established: %s", err)
}

// Close закрывает соединение с базой данных.
func (d *DB) Close() error {
	if err := d.Session.Close(); err != nil {
		return fmt.Errorf("can't close connection with db: %s", err)
	}

	return nil
}

type sqlScanner interface {
	Scan(dest ...interface{}) error
}
