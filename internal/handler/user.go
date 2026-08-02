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
	mux.HandleFunc("POST /api/users", h.CreateUser)
	mux.HandleFunc("GET /api/users", h.GetUsers)
	mux.HandleFunc("GET /api/users/{id}", h.GetUserByID)
	mux.HandleFunc("PUT /api/users", h.UpdateUserByID)
	mux.HandleFunc("DELETE /api/users", h.DeleteUserByID)

	mux.HandleFunc("POST /api/login", h.Login)
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

type LoginRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	ExpiresInSeconds *int64 `json:"expires_in_seconds,omitempty"` // Expiration time in seconds
}

type LoginResponse struct {
	Id        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
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

	user, err := h.apiConfig.Database.GetUserByEmail(r.Context(), request.Email)
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
	
	var expiration time.Duration
	if request.ExpiresInSeconds != nil && *request.ExpiresInSeconds > 0 && *request.ExpiresInSeconds <= 3600 {
		expiration = time.Duration(*request.ExpiresInSeconds) * time.Second
	} else {
		expiration = time.Hour
	}
	token, err := auth.MakeJWT(user.ID, h.apiConfig.Secret, expiration)
	if err != nil {
		h.apiConfig.Logger.Error("Failed to generate JWT", err)
		writeError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		Id:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		Token:     token,
	})
}

func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	// Implement the logic to get all users
}

func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	// Implement the logic to get a user by ID
}

func (h *UserHandler) UpdateUserByID(w http.ResponseWriter, r *http.Request) {
	// Implement the logic to update a user by ID
}

func (h *UserHandler) DeleteUserByID(w http.ResponseWriter, r *http.Request) {
	// Implement the logic to delete a user by ID
}
