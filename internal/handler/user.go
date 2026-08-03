package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/auth"
	"github.com/thalesraymond/go-http-server/internal/database"
)

type UserHandler struct {
	apiConfig *ApiConfig
	store     userStore
}

type userStore interface {
	CreateUser(ctx context.Context, arg database.CreateUserParams) (database.User, error)
	UpdateUser(ctx context.Context, arg database.UpdateUserParams) (database.User, error)
}

func NewUserHandler(apiConfig *ApiConfig) *UserHandler {
	return &UserHandler{apiConfig: apiConfig, store: apiConfig.Database}
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/users", h.CreateUser)
	mux.Handle("PUT /api/users", h.apiConfig.RequireAuth(h.UpdateUserByID))
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
		h.apiConfig.Logger.Error("Invalid request payload", err)
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	hashedPassword, err := auth.HashPassword(userRequestData.Password)
	if err != nil {
		h.apiConfig.Logger.Error("Failed to hash password", err)
		writeError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	createdUser, err := h.store.CreateUser(r.Context(), database.CreateUserParams{
		ID:             uuid.New(),
		Email:          userRequestData.Email,
		HashedPassword: hashedPassword,
	})

	if err != nil {
		h.apiConfig.Logger.Error("Failed to create user", err)
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
		h.apiConfig.Logger.Error("Invalid request payload", err)
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if updateUserRequestData.Email == "" || updateUserRequestData.Password == "" {
		h.apiConfig.Logger.Error("Email and password are required", nil)
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	hashedPassword, err := auth.HashPassword(updateUserRequestData.Password)
	if err != nil {
		h.apiConfig.Logger.Error("Failed to hash password", err)
		writeError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	updatedUser, err := h.store.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:             userID,
		Email:          updateUserRequestData.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		h.apiConfig.Logger.Error("Failed to update user", err)
		writeError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(updatedUser))
}
