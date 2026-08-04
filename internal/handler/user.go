package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/auth"
	"github.com/thalesraymond/go-http-server/internal/database"
)

var ErrCredentialsRequired = errors.New("email and password are required")

func validateUserCredentials(email, password string) error {
	if email == "" || password == "" {
		return ErrCredentialsRequired
	}
	return nil
}

type UserHandler struct {
	store         userStore
	logger        Logger
	authenticator auth.Authenticator
	handshake     *AuthHandshake
}

type userStore interface {
	CreateUser(ctx context.Context, arg database.CreateUserParams) (database.User, error)
	UpdateUser(ctx context.Context, arg database.UpdateUserParams) (database.User, error)
}

func NewUserHandler(store userStore, logger Logger, authenticator auth.Authenticator, handshake *AuthHandshake) *UserHandler {
	return &UserHandler{store: store, logger: logger, authenticator: authenticator, handshake: handshake}
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/users", h.CreateUser)
	mux.Handle("PUT /api/users", h.handshake.RequireAuth(h.UpdateUserByID))
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

func toUserResponse(user database.User) UserResponse {
	return UserResponse{
		ID:          user.ID,
		Email:       user.Email,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		IsChirpyRed: user.IsChirpyRed,
	}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var userRequestData CreateUserRequest

	if err := decodeJSON(w, r, &userRequestData); err != nil {
		h.logger.Error("Invalid request payload", err)
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := validateUserCredentials(userRequestData.Email, userRequestData.Password); err != nil {
		h.logger.Error("Email and password are required", err)
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	hashedPassword, err := h.authenticator.HashPassword(userRequestData.Password)
	if err != nil {
		h.logger.Error("Failed to hash password", err)
		writeError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	createdUser, err := h.store.CreateUser(r.Context(), database.CreateUserParams{
		ID:             uuid.New(),
		Email:          userRequestData.Email,
		HashedPassword: hashedPassword,
	})

	if err != nil {
		h.logger.Error("Failed to create user", err)
		writeError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	createdUserDTO := toUserResponse(createdUser)

	writeJSON(w, http.StatusCreated, createdUserDTO)
}

type UpdateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *UserHandler) UpdateUserByID(w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	var updateUserRequestData UpdateUserRequest
	if err := decodeJSON(w, r, &updateUserRequestData); err != nil {
		h.logger.Error("Invalid request payload", err)
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := validateUserCredentials(updateUserRequestData.Email, updateUserRequestData.Password); err != nil {
		h.logger.Error("Email and password are required", err)
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	hashedPassword, err := h.authenticator.HashPassword(updateUserRequestData.Password)
	if err != nil {
		h.logger.Error("Failed to hash password", err)
		writeError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	updatedUser, err := h.store.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:             userID,
		Email:          updateUserRequestData.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		h.logger.Error("Failed to update user", err)
		writeError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(updatedUser))
}
