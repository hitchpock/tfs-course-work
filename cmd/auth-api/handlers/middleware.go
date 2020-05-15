package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"gitlab.com/hitchpock/tfs-course-work/internal/session"
)

type idKey struct{}

type tokenKey struct{}

// getParamID проверяет URL на валидный id.
func (h *Handler) getParamID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paramID := chi.URLParam(r, "id")
		ID, err := strconv.Atoi(paramID)

		if err != nil || ID <= 0 {
			responseJSON := []byte(errorJSON("invalid id"))

			h.logger.Warnf("invalid id: %s", paramID)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			if _, e := w.Write(responseJSON); e != nil {
				h.logger.Warnf("unable to write responseJSON: %s", err)
				sendError(w, "error on server", http.StatusInternalServerError)
				return
			}

			return
		}

		ctx := context.WithValue(r.Context(), idKey{}, ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authentication проверяет валиден ли токен аунтификации из заголовков.
func (h *Handler) authentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetReqID(r.Context())
		remoteAddr := r.RemoteAddr
		value := r.Header.Get("Authorization")

		auth := strings.Split(value, " ")
		if len(auth) < 2 { //nolint:gomnd
			h.logger.Warnw("scheme not found", "error", value, "trackingID", reqID, "RealIP", remoteAddr)
			sendError(w, "scheme not found", http.StatusBadRequest)
			return
		}

		scheme, token := auth[0], auth[1]

		if token == "" {
			h.logger.Warnw("token not found", "trackingID", reqID, "RealIP", remoteAddr)
			sendError(w, "token not found", http.StatusUnauthorized)
			return
		}

		if scheme != "Bearer" {
			h.logger.Warnw("the authentication scheme is not supported", "error", scheme, "trackingID", reqID, "RealIP", remoteAddr)
			sendError(w, "the authentication scheme is not supported", http.StatusBadRequest)
			return
		}

		sessionToken, err := session.DecodeToken(token)
		if err != nil {
			h.logger.Warnw("invalid token", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
			sendError(w, "invalid token", http.StatusBadRequest)
			return
		}

		sessionToken.SessionID = token

		sessionStorage, err := h.sessionStorage.FindByUserID(sessionToken.UserID)
		if err != nil {
			h.logger.Warnw("session not found", "error", err, "trackingID", reqID, "RealIP", remoteAddr)
			sendError(w, "session not found", http.StatusNotFound)
			return
		}

		if !sessionStorage.Equal(sessionToken) {
			h.logger.Warnw("invalid token", "error", "session not equal", "trackingID", reqID, "RealIP", remoteAddr)
			sendError(w, "invalid token", http.StatusUnauthorized)
			return
		}

		if !(sessionToken.CreatedAt.Before(time.Now()) && sessionToken.ValidUntil.After(time.Now())) {
			h.logger.Warnw("session time is over", "trackingID", reqID, "RealIP", remoteAddr)
			sendError(w, "session time is over", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), tokenKey{}, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) authorization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetReqID(r.Context())
		remoteAddr := r.RemoteAddr
		ID := r.Context().Value(idKey{}).(int)
		token := r.Context().Value(tokenKey{}).(string)
		sessionToken, _ := session.DecodeToken(token)

		if sessionToken.UserID != ID {
			h.logger.Warnw("user have no permission", "userID", sessionToken.UserID, "trackingID", reqID, "RealIP", remoteAddr)
			sendError(w, "you have no permission", http.StatusForbidden)

			return
		}

		next.ServeHTTP(w, r)
	})
}
