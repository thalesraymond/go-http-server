package handler

import (
	"context"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/database"
)

// MockDatabase implements database.Querier for testing
type MockDatabase struct {
	CreateChirpFn         func(ctx context.Context, arg database.CreateChirpParams) (database.Chirp, error)
	CreateUserFn          func(ctx context.Context, arg database.CreateUserParams) (database.User, error)
	DeleteAllUsersFn      func(ctx context.Context) error
	GetUserByEmailFn      func(ctx context.Context, email string) (database.User, error)
	GetAllChirpsFn        func(ctx context.Context) ([]database.Chirp, error)
	GetChirpByIDFn        func(ctx context.Context, id uuid.UUID) (database.Chirp, error)
	CreateRefreshTokenFn  func(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error)
	GetRefreshTokenByIDFn func(ctx context.Context, token string) (database.RefreshToken, error)
	RevokeRefreshTokenFn  func(ctx context.Context, token string) error
}

func (m *MockDatabase) CreateChirp(ctx context.Context, arg database.CreateChirpParams) (database.Chirp, error) {
	if m.CreateChirpFn != nil {
		return m.CreateChirpFn(ctx, arg)
	}
	return database.Chirp{}, nil
}

func (m *MockDatabase) CreateUser(ctx context.Context, arg database.CreateUserParams) (database.User, error) {
	if m.CreateUserFn != nil {
		return m.CreateUserFn(ctx, arg)
	}
	return database.User{}, nil
}

func (m *MockDatabase) DeleteAllUsers(ctx context.Context) error {
	if m.DeleteAllUsersFn != nil {
		return m.DeleteAllUsersFn(ctx)
	}
	return nil
}

func (m *MockDatabase) GetUserByEmail(ctx context.Context, email string) (database.User, error) {
	if m.GetUserByEmailFn != nil {
		return m.GetUserByEmailFn(ctx, email)
	}
	return database.User{}, nil
}

func (m *MockDatabase) GetAllChirps(ctx context.Context) ([]database.Chirp, error) {
	if m.GetAllChirpsFn != nil {
		return m.GetAllChirpsFn(ctx)
	}
	return []database.Chirp{}, nil
}

func (m *MockDatabase) GetChirpByID(ctx context.Context, id uuid.UUID) (database.Chirp, error) {
	if m.GetChirpByIDFn != nil {
		return m.GetChirpByIDFn(ctx, id)
	}
	return database.Chirp{}, nil
}

func (m *MockDatabase) CreateRefreshToken(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error) {
	if m.CreateRefreshTokenFn != nil {
		return m.CreateRefreshTokenFn(ctx, arg)
	}
	return database.RefreshToken{}, nil
}

func (m *MockDatabase) GetRefreshTokenByID(ctx context.Context, token string) (database.RefreshToken, error) {
	if m.GetRefreshTokenByIDFn != nil {
		return m.GetRefreshTokenByIDFn(ctx, token)
	}
	return database.RefreshToken{}, nil
}

func (m *MockDatabase) RevokeRefreshToken(ctx context.Context, token string) error {
	if m.RevokeRefreshTokenFn != nil {
		return m.RevokeRefreshTokenFn(ctx, token)
	}
	return nil
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
