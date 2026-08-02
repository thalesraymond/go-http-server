package handler

import (
	"context"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/database"
)

// noopQuerier only exists to satisfy ApiConfig.Database in tests that don't care
// about sqlc calls directly.
type noopQuerier struct{}

func (n *noopQuerier) CreateChirp(_ context.Context, _ database.CreateChirpParams) (database.Chirp, error) {
	return database.Chirp{}, nil
}

func (n *noopQuerier) CreateRefreshToken(_ context.Context, _ database.CreateRefreshTokenParams) (database.RefreshToken, error) {
	return database.RefreshToken{}, nil
}

func (n *noopQuerier) CreateUser(_ context.Context, _ database.CreateUserParams) (database.User, error) {
	return database.User{}, nil
}

func (n *noopQuerier) DeleteAllUsers(_ context.Context) error {
	return nil
}

func (n *noopQuerier) DeleteChirpByID(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (n *noopQuerier) GetAllChirps(_ context.Context) ([]database.Chirp, error) {
	return []database.Chirp{}, nil
}

func (n *noopQuerier) GetChirpByID(_ context.Context, _ uuid.UUID) (database.Chirp, error) {
	return database.Chirp{}, nil
}

func (n *noopQuerier) GetRefreshTokenByID(_ context.Context, _ string) (database.RefreshToken, error) {
	return database.RefreshToken{}, nil
}

func (n *noopQuerier) GetUserByEmail(_ context.Context, _ string) (database.User, error) {
	return database.User{}, nil
}

func (n *noopQuerier) RevokeRefreshToken(_ context.Context, _ string) error {
	return nil
}

func (n *noopQuerier) UpdateUser(_ context.Context, _ database.UpdateUserParams) (database.User, error) {
	return database.User{}, nil
}

type mockUserStore struct {
	createUserFn func(ctx context.Context, arg database.CreateUserParams) (database.User, error)
	updateUserFn func(ctx context.Context, arg database.UpdateUserParams) (database.User, error)
}

func (m *mockUserStore) CreateUser(ctx context.Context, arg database.CreateUserParams) (database.User, error) {
	if m.createUserFn != nil {
		return m.createUserFn(ctx, arg)
	}
	return database.User{}, nil
}

func (m *mockUserStore) UpdateUser(ctx context.Context, arg database.UpdateUserParams) (database.User, error) {
	if m.updateUserFn != nil {
		return m.updateUserFn(ctx, arg)
	}
	return database.User{}, nil
}

type mockLoginStore struct {
	getUserByEmailFn      func(ctx context.Context, email string) (database.User, error)
	createRefreshTokenFn  func(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error)
	getRefreshTokenByIDFn func(ctx context.Context, token string) (database.RefreshToken, error)
	revokeRefreshTokenFn  func(ctx context.Context, token string) error
}

func (m *mockLoginStore) GetUserByEmail(ctx context.Context, email string) (database.User, error) {
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(ctx, email)
	}
	return database.User{}, nil
}

func (m *mockLoginStore) CreateRefreshToken(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error) {
	if m.createRefreshTokenFn != nil {
		return m.createRefreshTokenFn(ctx, arg)
	}
	return database.RefreshToken{}, nil
}

func (m *mockLoginStore) GetRefreshTokenByID(ctx context.Context, token string) (database.RefreshToken, error) {
	if m.getRefreshTokenByIDFn != nil {
		return m.getRefreshTokenByIDFn(ctx, token)
	}
	return database.RefreshToken{}, nil
}

func (m *mockLoginStore) RevokeRefreshToken(ctx context.Context, token string) error {
	if m.revokeRefreshTokenFn != nil {
		return m.revokeRefreshTokenFn(ctx, token)
	}
	return nil
}

type mockChirpStore struct {
	createChirpFn     func(ctx context.Context, arg database.CreateChirpParams) (database.Chirp, error)
	getAllChirpsFn    func(ctx context.Context) ([]database.Chirp, error)
	getChirpByIDFn    func(ctx context.Context, id uuid.UUID) (database.Chirp, error)
	deleteChirpByIDFn func(ctx context.Context, id uuid.UUID) error
}

func (m *mockChirpStore) CreateChirp(ctx context.Context, arg database.CreateChirpParams) (database.Chirp, error) {
	if m.createChirpFn != nil {
		return m.createChirpFn(ctx, arg)
	}
	return database.Chirp{}, nil
}

func (m *mockChirpStore) GetAllChirps(ctx context.Context) ([]database.Chirp, error) {
	if m.getAllChirpsFn != nil {
		return m.getAllChirpsFn(ctx)
	}
	return []database.Chirp{}, nil
}

func (m *mockChirpStore) GetChirpByID(ctx context.Context, id uuid.UUID) (database.Chirp, error) {
	if m.getChirpByIDFn != nil {
		return m.getChirpByIDFn(ctx, id)
	}
	return database.Chirp{}, nil
}

func (m *mockChirpStore) DeleteChirpByID(ctx context.Context, id uuid.UUID) error {
	if m.deleteChirpByIDFn != nil {
		return m.deleteChirpByIDFn(ctx, id)
	}
	return nil
}

func newTestUserHandler(store *mockUserStore) *UserHandler {
	h := NewUserHandler(newTestApiConfig())
	if store != nil {
		h.store = store
	}
	return h
}

func newTestLoginHandler(store *mockLoginStore) *LoginHandler {
	h := NewLoginHandler(newTestApiConfig())
	if store != nil {
		h.store = store
	}
	return h
}

func newTestChirpHandler(store *mockChirpStore) (*ApiConfig, *ChirpHandler) {
	cfg := newTestApiConfig()
	h := NewChirpHandler(cfg)
	if store != nil {
		h.store = store
	}
	return cfg, h
}

// MockLogger implements Logger interface for testing
type MockLogger struct {
	ErrorFn func(msg string, err error)
	InfoFn  func(msg string)
	DebugFn func(msg string)
	WarnFn  func(msg string)
}

func (m *MockLogger) Error(msg string, err error) {
	if m.ErrorFn != nil {
		m.ErrorFn(msg, err)
	}
}

func (m *MockLogger) Info(msg string) {
	if m.InfoFn != nil {
		m.InfoFn(msg)
	}
}

func (m *MockLogger) Debug(msg string) {
	if m.DebugFn != nil {
		m.DebugFn(msg)
	}
}

func (m *MockLogger) Warn(msg string) {
	if m.WarnFn != nil {
		m.WarnFn(msg)
	}
}
