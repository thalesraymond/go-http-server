package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/auth"
	"github.com/thalesraymond/go-http-server/internal/database"
)

type UserHandler struct {
	apiConfig *ApiConfig
}

func NewUserHandler(apiConfig *ApiConfig) *UserHandler {
	return &UserHandler{apiConfig: apiConfig}
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/users", h.HandlerCreateUser)
	mux.HandleFunc("GET /api/users", h.HandlerGetUsers)
	mux.HandleFunc("GET /api/users/{id}", h.HandlerGetUserByID)
	mux.HandleFunc("PUT /api/users", h.HandlerUpdateUserByID)
	mux.HandleFunc("DELETE /api/users", h.HandlerDeleteUserByID)
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toUserResponse(user database.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func (h *UserHandler) HandlerCreateUser(w http.ResponseWriter, r *http.Request) {
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

	createdUser, err := h.apiConfig.Database.CreateUser(r.Context(), database.CreateUserParams{
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

func (h *UserHandler) HandlerGetUsers(w http.ResponseWriter, r *http.Request) {
	// Implement the logic to get all users
}

func (h *UserHandler) HandlerGetUserByID(w http.ResponseWriter, r *http.Request) {
	// Implement the logic to get a user by ID
}

func (h *UserHandler) HandlerUpdateUserByID(w http.ResponseWriter, r *http.Request) {
	// Implement the logic to update a user by ID
}

func (h *UserHandler) HandlerDeleteUserByID(w http.ResponseWriter, r *http.Request) {
	// Implement the logic to delete a user by ID
}
