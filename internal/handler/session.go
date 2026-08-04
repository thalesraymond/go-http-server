package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/auth"
	"github.com/thalesraymond/go-http-server/internal/database"
	"github.com/thalesraymond/go-http-server/internal/session"
)

type SessionHandler struct {
	store         sessionUserStore
	logger        Logger
	authenticator auth.Authenticator
	session       *session.Session
}

type sessionUserStore interface {
	GetUserByEmail(ctx context.Context, email string) (database.User, error)
}

func NewSessionHandler(store sessionUserStore, logger Logger, authenticator auth.Authenticator, session *session.Session) *SessionHandler {
	return &SessionHandler{store: store, logger: logger, authenticator: authenticator, session: session}
}

func (h *SessionHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/login", h.Login)
	mux.HandleFunc("POST /api/refresh", h.GetRefreshToken)
	mux.HandleFunc("POST /api/revoke", h.RevokeRefreshToken)
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}

func (h *SessionHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request LoginRequest
	err := decodeJSON(w, r, &request)

	if err != nil {
		h.logger.Error("Invalid request payload", err)
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if request.Email == "" || request.Password == "" {
		h.logger.Error("Email and password are required", nil)
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := h.store.GetUserByEmail(r.Context(), request.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "Incorrect email or password")
			return
		}

		h.logger.Error("Failed to get user by email", err)
		writeError(w, http.StatusInternalServerError, "Failed to get user by email")
		return
	}

	match, err := h.authenticator.CheckPasswordHash(request.Password, user.HashedPassword)
	if err != nil {
		h.logger.Error("Failed to check password hash", err)
		writeError(w, http.StatusInternalServerError, "Failed to check password hash")
		return
	}

	if !match {
		h.logger.Error("Incorrect password", nil)
		writeError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	accessToken, refreshToken, err := h.session.Start(r.Context(), user.ID)
	if err != nil {
		h.writeSessionError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		IsChirpyRed:  user.IsChirpyRed,
		Token:        accessToken,
		RefreshToken: refreshToken,
	})
}

func (h *SessionHandler) GetRefreshToken(w http.ResponseWriter, r *http.Request) {
	accessToken, err := h.session.Renew(r.Context(), r)
	if err != nil {
		h.writeSessionError(w, err)
		return
	}

	type response struct {
		Token string `json:"token"`
	}

	writeJSON(w, http.StatusOK, response{
		Token: accessToken,
	})
}

func (h *SessionHandler) RevokeRefreshToken(w http.ResponseWriter, r *http.Request) {
	err := h.session.End(r.Context(), r)
	if err != nil {
		h.writeSessionError(w, err)
		return
	}

	writeNoContent(w)
}

func (h *SessionHandler) writeSessionError(w http.ResponseWriter, err error) {
	if errors.Is(err, session.ErrRefreshTokenInvalid) ||
		errors.Is(err, session.ErrRefreshTokenRevoked) ||
		errors.Is(err, session.ErrRefreshTokenExpired) {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	h.logger.Error("Session operation failed", err)
	writeError(w, http.StatusInternalServerError, "Internal Server Error")
}
