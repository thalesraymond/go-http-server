package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/auth"
	"github.com/thalesraymond/go-http-server/internal/database"
)

type LoginHandler struct {
	apiConfig *ApiConfig
	store     loginStore
}

type loginStore interface {
	GetUserByEmail(ctx context.Context, email string) (database.User, error)
	CreateRefreshToken(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error)
	GetRefreshTokenByID(ctx context.Context, token string) (database.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error
}

func NewLoginHandler(apiConfig *ApiConfig) *LoginHandler {
	return &LoginHandler{apiConfig: apiConfig, store: apiConfig.Database}
}

func (h *LoginHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/login", h.Login)
	mux.HandleFunc("POST /api/refresh", h.GetRefreshToken)
	mux.HandleFunc("POST /api/revoke", h.RevokeRefreshToken)
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Id           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}

func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request LoginRequest
	err := decodeJSON(w, r, &request)

	if err != nil {
		h.apiConfig.Logger.Error("Invalid request payload", err)
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if request.Email == "" || request.Password == "" {
		h.apiConfig.Logger.Error("Email and password are required", nil)
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := h.store.GetUserByEmail(r.Context(), request.Email)
	if err != nil {
		h.apiConfig.Logger.Error("Failed to get user by email", err)
		writeError(w, http.StatusInternalServerError, "Incorrect email or password")
		return
	}

	match, err := auth.CheckPasswordHash(request.Password, user.HashedPassword)
	if err != nil {
		h.apiConfig.Logger.Error("Failed to check password hash", err)
		writeError(w, http.StatusInternalServerError, "Incorrect email or password")
		return
	}

	if !match {
		h.apiConfig.Logger.Error("Incorrect password", nil)
		writeError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	token, err := auth.MakeJWT(user.ID, h.apiConfig.Secret, time.Hour)

	if err != nil {
		h.apiConfig.Logger.Error("Failed to generate JWT", err)
		writeError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	refreshToken, err := h.store.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     auth.MakeRefreshToken(),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	if err != nil {
		h.apiConfig.Logger.Error("Failed to create refresh token", err)
		writeError(w, http.StatusInternalServerError, "Failed to create refresh token")
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		Id:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        token,
		RefreshToken: refreshToken.Token,
	})
}

func (h *LoginHandler) GetRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshTokenValue, err := auth.GetBearerToken(r.Header)

	if err != nil {
		h.apiConfig.Logger.Error("Invalid Authorization header", err)
		writeError(w, http.StatusUnauthorized, "Invalid Authorization header")
		return
	}

	refreshToken, err := h.store.GetRefreshTokenByID(r.Context(), refreshTokenValue)

	if err != nil {
		h.apiConfig.Logger.Error("Failed to get refresh token", err)
		writeError(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	if refreshToken.RevokedAt.Valid {
		h.apiConfig.Logger.Error("Refresh token revoked", nil)
		writeError(w, http.StatusUnauthorized, "Refresh token revoked")
		return
	}

	if refreshToken.ExpiresAt.Before(time.Now()) {
		h.apiConfig.Logger.Error("Refresh token expired", nil)
		writeError(w, http.StatusUnauthorized, "Refresh token expired")
		return
	}

	token, err := auth.MakeJWT(refreshToken.UserID, h.apiConfig.Secret, time.Hour)
	if err != nil {
		h.apiConfig.Logger.Error("Failed to generate JWT", err)
		writeError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	type response struct {
		Token string `json:"token"`
	}

	writeJSON(w, http.StatusOK, response{
		Token: token,
	})
}

func (h *LoginHandler) RevokeRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshTokenValue, err := auth.GetBearerToken(r.Header)

	if err != nil {
		h.apiConfig.Logger.Error("Invalid Authorization header", err)
		writeError(w, http.StatusUnauthorized, "Invalid Authorization header")
		return
	}

	err = h.store.RevokeRefreshToken(r.Context(), refreshTokenValue)
	if err != nil {
		h.apiConfig.Logger.Error("Failed to remove refresh token", err)
		writeError(w, http.StatusInternalServerError, "Failed to remove refresh token")
		return
	}

	writeNoContent(w)
}
