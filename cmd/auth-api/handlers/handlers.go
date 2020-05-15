package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"gitlab.com/hitchpock/tfs-course-work/internal/robot"
	"gitlab.com/hitchpock/tfs-course-work/internal/session"
	"gitlab.com/hitchpock/tfs-course-work/internal/user"
	"gitlab.com/hitchpock/tfs-course-work/pkg/log"
)

const textHTML = "text/html"

// Handler структура хэндлера сервиса.
type Handler struct {
	logger         log.Logger
	sessionStorage session.Storage
	userStorage    user.Storage
	robotStorage   robot.Storage
	wsocket        *WSClients
}

// NewHandler возвращает указатель на новый хэндлер.
func NewHandler(logger log.Logger, sessions session.Storage, users user.Storage, robots robot.Storage, socket *WSClients) *Handler {
	return &Handler{
		logger:         logger,
		sessionStorage: sessions,
		userStorage:    users,
		robotStorage:   robots,
		wsocket:        socket,
	}
}

// Routes возвращает указатель на роутинг сервиса.
func (h *Handler) Routes() chi.Router {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.SetHeader("Content-type", "application/json"))
	//router.Use(middleware.Timeout(100 * time.Millisecond))

	router.Route("/api/v1", func(router chi.Router) {
		router.Post("/signup", h.SignUp)
		router.Post("/signin", h.SignIn)

		router.Route("/users/{id}", func(router chi.Router) {
			router.Use(h.getParamID, h.authentication, h.authorization)

			router.Get("/robots", h.UserRobots)
			router.Put("/", h.UpdateUser)
			router.Get("/", h.GetUser)
		})

		router.Route("/", func(router chi.Router) {
			router.Use(h.authentication)

			router.Get("/robots", h.CatalogRobots)
			router.Post("/robot", h.CreateRobot)
		})

		router.Route("/robot/{id}", func(router chi.Router) {
			router.Use(h.getParamID, h.authentication)

			router.Put("/favourite", h.FavouriteRobot) //nolint:misspell
			router.Put("/activate", h.ActivateRobot)
			router.Put("/deactivate", h.DeactivateRobot)
			router.Get("/", h.RobotDetails)
			router.Delete("/", h.DeleteRobot)
		})

		router.HandleFunc("/wsrobotdetail", h.wsocket.WSRobotDeltail)
	})

	return router
}

// SignUp регистрирует пользователя в сервисе.
func (h *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	remoteAddr := r.RemoteAddr

	u, err := getUserFromBody(r.Body)
	if err != nil {
		h.logger.Warnw("func getUserFromBody is crashed", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "invalid input", http.StatusBadRequest)

		return
	}
	defer r.Body.Close()

	if err = h.userStorage.Create(u); err != nil {
		h.logger.Warnw("func Create return with error", "error", err, "trackingID", reqID, "RealIP", remoteAddr)

		responseJSON := []byte(errorJSON(err.Error()))

		w.WriteHeader(http.StatusConflict)

		if _, e := w.Write(responseJSON); e != nil {
			h.logger.Warnw("unable to write responseJson", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
			sendError(w, "error on server", http.StatusInternalServerError)

			return
		}

		return
	}

	h.logger.Infow("signup", "user", u.Email, "trackingID", reqID, "RealIP", remoteAddr)
	w.WriteHeader(http.StatusCreated)
}

// SignIn аунтифицирует пользователя в системе.
func (h *Handler) SignIn(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	remoteAddr := r.RemoteAddr

	u, err := getSignInDataFromBody(r.Body)
	if err != nil {
		h.logger.Warnw("func getSignInData is crashed", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "invalid input", http.StatusBadRequest)

		return
	}
	defer r.Body.Close()

	userStorage, err := h.userStorage.FindByEmail(u.Email)
	if err != nil {
		userStorage = &user.User{}
	}

	ch := CheckPasswordHash(userStorage.Password, u.Password)
	if err != nil || !ch {
		responseJSON := []byte(errorJSON("incorrect email or password"))

		w.WriteHeader(http.StatusBadRequest)

		if _, e := w.Write(responseJSON); e != nil {
			h.logger.Warnw("unable to write responseJSON", "error", e, "trackingID", reqID, "RealIP", remoteAddr)
			sendError(w, "error on server", http.StatusInternalServerError)
		}

		return
	}

	h.logger.Infow("signin", "user", userStorage.Email, "trackingID", reqID, "RealIP", remoteAddr)

	ses := session.NewSession(userStorage.ID)
	if err = h.sessionStorage.Create(ses); err != nil {
		h.logger.Warnw("can't add session in storage", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)

		return
	}

	token := session.BearerToken{Token: ses.SessionID}

	tokenJSON, err := json.Marshal(token)
	if err != nil {
		h.logger.Warnw("marhsal tokenJSON is crashed", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)

	if _, err = w.Write(tokenJSON); err != nil {
		h.logger.Warnw("unable to write tokenJSON", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)
	}
}

// UpdateUser обновляет авторизованного пользователя.
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	remoteAddr := r.RemoteAddr
	token := r.Context().Value(tokenKey{}).(string)
	sessionToken, _ := session.DecodeToken(token)

	userRequest, err := getUserFromBody(r.Body)
	if err != nil {
		h.logger.Warnw("func getUserFromBody return with error", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "invalid input", http.StatusBadRequest)

		return
	}
	defer r.Body.Close()

	userSession, err := h.userStorage.FindByID(sessionToken.UserID)
	if err != nil {
		responseJSON := []byte(errorJSON(err.Error()))

		w.WriteHeader(http.StatusNotFound)

		if _, err = w.Write(responseJSON); err != nil {
			h.logger.Warnw("unable to write tokenJSON", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
			sendError(w, "error on server", http.StatusInternalServerError)
		}

		return
	}

	if u, _ := h.userStorage.FindByEmail(userRequest.Email); u != nil && u.ID != userSession.ID {
		sendError(w, "email is already in use", http.StatusBadRequest)
		return
	}

	userSession.Update(userRequest)

	if err = h.userStorage.Update(userSession); err != nil {
		h.logger.Infow("func Update user in storage is crashed", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)

		return
	}

	h.logger.Infow("update user", "user", userSession.Email, "trackingID", reqID, "RealIP", remoteAddr)

	userResponseJSON, err := userSession.MarshalJSON()
	if err != nil {
		h.logger.Warnw("marhsal userResponseJSON is crashed", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)

	if _, err = w.Write(userResponseJSON); err != nil {
		h.logger.Warnw("unable to write userResponseJSON", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)
	}
}

// GetUser получает сущность пользователя.
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	remoteAddr := r.RemoteAddr
	userID := r.Context().Value(idKey{}).(int)

	u, err := h.userStorage.FindByID(userID)
	if err != nil {
		responseJSON := []byte(errorJSON(err.Error()))

		w.WriteHeader(http.StatusNotFound)

		if _, err = w.Write(responseJSON); err != nil {
			h.logger.Warnw("unable to write tokenJSON", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
			sendError(w, "error on server", http.StatusInternalServerError)
		}

		return
	}

	userResponseJSON, err := u.MarshalJSON()
	if err != nil {
		h.logger.Warnw("marhsal userResponseJSON is crashed", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)

	if _, err = w.Write(userResponseJSON); err != nil {
		h.logger.Warnw("unable to write userResponseJSON", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)
	}
}

// CreateRobot создает робота из запроса.
func (h *Handler) CreateRobot(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	remoteAddr := r.RemoteAddr

	token := r.Context().Value(tokenKey{}).(string)
	sessionToken, _ := session.DecodeToken(token)

	robotRequest, err := getRobotFromBody(r.Body)
	if err != nil {
		h.logger.Warnw("func getRobotFromRequest return with error", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "invalid input", http.StatusBadRequest)

		return
	}
	defer r.Body.Close()

	if sessionToken.UserID != robotRequest.OwnerUserID {
		h.logger.Warnw("user have no permission", "userID", sessionToken.UserID, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "you have no permission", http.StatusForbidden)

		return
	}

	robotRequest.FactYield = 0.0
	robotRequest.DealsCount = 0
	robotRequest.ParentRobotID = 0
	robotRequest.IsActive = false
	robotRequest.IsFavourite = false

	if err := h.robotStorage.Create(robotRequest); err != nil {
		h.logger.Warnw("func robotStorage.Create return with error", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "can't create robot", http.StatusInternalServerError)

		return
	}

	h.logger.Infow("create robot", "user", sessionToken.UserID, "trackingID", reqID, "ReadIP", remoteAddr)
	w.WriteHeader(http.StatusCreated)
}

// DeleteRobot выполняет SoftDelete робота по id.
func (h *Handler) DeleteRobot(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	remoteAddr := r.RemoteAddr
	robotID := r.Context().Value(idKey{}).(int)
	token := r.Context().Value(tokenKey{}).(string)
	sessionToken, _ := session.DecodeToken(token)

	robotStorage, err := h.robotStorage.FindByID(robotID)
	if err != nil {
		if errors.Is(err, robot.ErrNotFound) {
			h.logger.Warnw("robot not found", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
			sendError(w, "robot not found", http.StatusNotFound)

			return
		}

		h.logger.Warnw("func robotStorage.FindByID return with error", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusNotFound)

		return
	}

	if robotStorage.OwnerUserID != sessionToken.UserID {
		h.logger.Warnw("user have no permission", "userID", sessionToken.UserID, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "you have no permission", http.StatusForbidden)

		return
	}

	if err = h.robotStorage.SoftDelete(robotID); err != nil {
		h.logger.Warnw("func robotStorage.SoftDelete return with error", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)

		return
	}

	h.logger.Infow("'soft delete' robot", "userID", sessionToken.UserID)
	w.WriteHeader(http.StatusOK)
}

// UserRobots отправляет список неудаленных пользовательских роботов.
func (h *Handler) UserRobots(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	remoteAddr := r.RemoteAddr
	userID := r.Context().Value(idKey{}).(int)

	robots, err := h.robotStorage.FindActivatedByUserID(userID)
	if err != nil {
		h.logger.Warnw("func robotStorage.FindActivatedByUserID return with error", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)

	if r.Header.Get("Accept") == textHTML {
		w.Header().Set("Content-type", textHTML)
		renderTemplate(w, "listrobots", "base", robots)

		return
	}

	robotsJSON, err := json.Marshal(robots)
	if err != nil {
		h.logger.Warnw("unable to marshal robots", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)

		return
	}

	if _, err = w.Write(robotsJSON); err != nil {
		h.logger.Warnw("unable to write robotsJSON", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)
	}
}

// CatalogRobots возвращает отсортированный по query-параметрам список роботов.
func (h *Handler) CatalogRobots(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	remoteAddr := r.RemoteAddr

	ticker := r.URL.Query().Get("ticker")
	userID := r.URL.Query().Get("user")

	robots, err := h.robotStorage.Filter(ticker, userID)
	if err != nil {
		if errors.Is(err, robot.ErrInvalidID) {
			h.logger.Warnw("invalid id", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
			sendError(w, "invalid id", http.StatusInternalServerError)

			return
		}

		h.logger.Warnw("func robotStorage.Filter return with error", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)

	if r.Header.Get("Accept") == "text/html" {
		w.Header().Set("Content-type", "text/html")
		renderTemplate(w, "listrobots", "base", robots)

		return
	}

	robotsJSON, err := json.Marshal(robots)
	if err != nil {
		h.logger.Warnw("unable to marshal robots", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)

		return
	}

	if _, err = w.Write(robotsJSON); err != nil {
		h.logger.Warnw("unable to write robotsJSON", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)
	}
}

// Добаляет робота в список избранных.
func (h *Handler) FavouriteRobot(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	remoteAddr := r.RemoteAddr
	robotID := r.Context().Value(idKey{}).(int)
	token := r.Context().Value(tokenKey{}).(string)
	sessionToken, _ := session.DecodeToken(token)

	if err := h.robotStorage.FavouriteRobot(robotID, sessionToken.UserID); err != nil {
		if errors.Is(err, robot.ErrNotFound) {
			h.logger.Warnw("robot not found", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
			sendError(w, "robot not found", http.StatusNotFound)

			return
		}

		h.logger.Warnw("func robotStorage.FavouriteRobot return with error", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)

		return
	}

	h.wsocket.Broadcast(robotID)
	w.WriteHeader(http.StatusOK)
}

// ActivateRobot активирует робота пользователя.
func (h *Handler) ActivateRobot(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	remoteAddr := r.RemoteAddr
	robotID := r.Context().Value(idKey{}).(int)
	token := r.Context().Value(tokenKey{}).(string)
	sessionToken, _ := session.DecodeToken(token)

	rob, err := h.robotStorage.FindByID(robotID)
	if err != nil {
		if errors.Is(err, robot.ErrNotFound) {
			h.logger.Warnw("robot not found", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
			sendError(w, "robot not found", http.StatusNotFound)

			return
		}

		h.logger.Warnw("func robotStorage.FavouriteRobot return with error", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)

		return
	}

	if rob.OwnerUserID != sessionToken.UserID {
		h.logger.Warnw("user have no permission", "userID", sessionToken.UserID, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "you have no permission", http.StatusForbidden)

		return
	}

	if rob.IsActive {
		h.logger.Warnw("robot already activated", "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "robot already activated", http.StatusBadRequest)

		return
	}

	if !rob.PlanStart.Valid || !rob.PlanEnd.Valid || (time.Now().After(rob.PlanStart.Time) && time.Now().Before(rob.PlanEnd.Time)) {
		h.logger.Warnw("time activated in range [plan_start:plan_end] or plan_start/plan_end is NULL", "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "time activated in range [plan_start:plan_end] or plan_start/plan_end is NULL", http.StatusBadRequest)

		return
	}

	if err = h.robotStorage.ActivateRobot(robotID); err != nil {
		h.logger.Warnw("func robotStorage return with error", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)

		return
	}

	h.wsocket.Broadcast(rob.RobotID)
	w.WriteHeader(http.StatusOK)
}

// DeactivateRobot дективирует робота пользователя.
func (h *Handler) DeactivateRobot(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	remoteAddr := r.RemoteAddr
	robotID := r.Context().Value(idKey{}).(int)
	token := r.Context().Value(tokenKey{}).(string)
	sessionToken, _ := session.DecodeToken(token)

	rob, err := h.robotStorage.FindByID(robotID)
	if err != nil {
		if errors.Is(err, robot.ErrNotFound) {
			h.logger.Warnw("robot not found", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
			sendError(w, "robot not found", http.StatusNotFound)

			return
		}

		h.logger.Warnw("func robotStorage.FavouriteRobot return with error", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)

		return
	}

	if rob.OwnerUserID != sessionToken.UserID {
		h.logger.Warnw("user have no permission", "userID", sessionToken.UserID, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "you have no permission", http.StatusForbidden)

		return
	}

	if !rob.IsActive {
		h.logger.Warnw("robot already deactivated", "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "robot already deactivated", http.StatusBadRequest)

		return
	}

	if !rob.PlanStart.Valid || !rob.PlanEnd.Valid || (time.Now().After(rob.PlanStart.Time) && time.Now().Before(rob.PlanEnd.Time)) {
		h.logger.Warnw("time deactivated in range [plan_start:plan_end] or plan_start/plan_end is NULL", "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "time deactivated in range [plan_start:plan_end] or plan_start/plan_end is NULL", http.StatusBadRequest)

		return
	}

	if err = h.robotStorage.DeactivateRobot(robotID); err != nil {
		h.logger.Warnw("func robotStorage.Deactivate return with error", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)

		return
	}

	h.wsocket.Broadcast(rob.RobotID)
	w.WriteHeader(http.StatusOK)
}

// RobotDetails возвращает json/html представление одного робота.
func (h *Handler) RobotDetails(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	remoteAddr := r.RemoteAddr
	robotID := r.Context().Value(idKey{}).(int)

	rob, err := h.robotStorage.FindByID(robotID)
	if err != nil {
		h.logger.Warnw("robot not found", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "robot not found", http.StatusNotFound)

		return
	}

	w.WriteHeader(http.StatusOK)

	if r.Header.Get("Accept") == textHTML {
		w.Header().Set("Content-type", "text/html")
		renderTemplate(w, "robotdetail", "base", rob)

		return
	}

	robotJSON, err := rob.MarshalJSON()
	if err != nil {
		h.logger.Warnw("unable to marshal robot", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)

		return
	}

	if _, err = w.Write(robotJSON); err != nil {
		h.logger.Warnw("unable to write robotJSON", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
		sendError(w, "error on server", http.StatusInternalServerError)
	}
}
