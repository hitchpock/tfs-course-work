package main

import (
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"gitlab.com/hitchpock/tfs-course-work/cmd/auth-api/handlers"
	"gitlab.com/hitchpock/tfs-course-work/cmd/auth-api/trading"
	"gitlab.com/hitchpock/tfs-course-work/internal/postgres"
	zp "gitlab.com/hitchpock/tfs-course-work/pkg/log"
	"google.golang.org/grpc"
)

const (
	port = ":8080"

	ReadTimeoutInt  = 2
	WriteTimeoutInt = 2

	URL             = "postgres://fintech:fintech@localhost:5432/fintech_go"
	ConnMaxLifetime = time.Minute
	MaxOpenConns    = 10
	MaxIdleConns    = 2
)

func main() {
	cfgDB := configDB()

	logger := zp.NewSugarLogger()
	defer logger.Sugar.Sync() // nolint:errcheck

	db, err := postgres.New(logger, cfgDB)
	if err != nil {
		logger.Fatalf("can't create db: %s", err)
	}

	defer handleCloser(logger, "db", db)

	if err = db.CheckConnection(); err != nil {
		logger.Fatalf("can't check connection: %s", err)
	}

	userStorage, err := postgres.NewUserStorage(db)
	if err != nil {
		logger.Fatalf("can't create user storage: %s", err)
	}

	defer handleCloser(logger, "userStorage", userStorage)

	sessionStorage, err := postgres.NewSessionStorage(db)
	if err != nil {
		logger.Fatalf("can't create session storage: %s", err)
	}

	defer handleCloser(logger, "sessionStorage", sessionStorage)

	robotStorage, err := postgres.NewRobotStorage(db)
	if err != nil {
		logger.Fatalf("can't create robot storage: %s", err)
	}

	defer handleCloser(logger, "robotStorage", robotStorage)

	conn, err := grpc.Dial("localhost:5000", grpc.WithInsecure())
	if err != nil {
		logger.Fatalf("can't create connect to grpc server: %s", err)
	}
	defer conn.Close()

	wsocket := handlers.NewWebsocket(robotStorage)
	handler := handlers.NewHandler(logger, sessionStorage, userStorage, robotStorage, wsocket)
	router := routes(handler)
	backgroundTrading := trading.NewProcess(conn, logger, robotStorage, wsocket)
	srv := configServer(router)

	go backgroundTrading.StartTrading()

	logger.Infof("Application is run on port %s", port)

	if err := srv.ListenAndServe(); err != nil {
		logger.Fatalf("HTTP server ListenAndServe: %s", err)
	}
}

func routes(h *handlers.Handler) *chi.Mux {
	r := chi.NewRouter()

	r.Mount("/", h.Routes())

	return r
}

func handleCloser(logger zp.Logger, resource string, closer io.Closer) {
	if err := closer.Close(); err != nil {
		logger.Warnf("can't close %q: %s", resource, err)
	}
}

func configDB() postgres.Config {
	cfgDB := postgres.Config{
		URL:             URL,
		ConnMaxLifetime: ConnMaxLifetime,
		MaxOpenConns:    MaxOpenConns,
		MaxIdleConns:    MaxIdleConns,
	}

	return cfgDB
}

func configServer(router http.Handler) *http.Server {
	srv := &http.Server{
		Addr:         port,
		ReadTimeout:  ReadTimeoutInt * time.Second,
		WriteTimeout: WriteTimeoutInt * time.Second,
		Handler:      router,
	}

	return srv
}
