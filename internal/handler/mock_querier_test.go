package handler

import (
	"context"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/database"
)

var _ database.Querier = (*mockQuerier)(nil)

// mockQuerier implements database.Querier with per-method stub functions.
// Every method records its arguments so tests can assert on them. Because
// the handler store interfaces (chirpStore, loginStore, userStore,
// polkaStore) are subsets of Querier, this one fake satisfies all of them.
type mockQuerier struct {
	createChirpFn         func(ctx context.Context, arg database.CreateChirpParams) (database.Chirp, error)
	getAllChirpsAscFn     func(ctx context.Context, authorID uuid.NullUUID) ([]database.Chirp, error)
	getAllChirpsDescFn    func(ctx context.Context, authorID uuid.NullUUID) ([]database.Chirp, error)
	getChirpByIDFn        func(ctx context.Context, id uuid.UUID) (database.Chirp, error)
	deleteChirpByIDFn     func(ctx context.Context, id uuid.UUID) error
	getUserByEmailFn      func(ctx context.Context, email string) (database.User, error)
	createRefreshTokenFn  func(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error)
	getRefreshTokenByIDFn func(ctx context.Context, token string) (database.RefreshToken, error)
	revokeRefreshTokenFn  func(ctx context.Context, token string) error
	createUserFn          func(ctx context.Context, arg database.CreateUserParams) (database.User, error)
	updateUserFn          func(ctx context.Context, arg database.UpdateUserParams) (database.User, error)
	updateChirpyRedFlagFn func(ctx context.Context, arg database.UpdateChirpyRedFlagParams) (database.User, error)
	deleteAllUsersFn      func(ctx context.Context) error

	createChirpParams         database.CreateChirpParams
	getAllChirpsAscAuthorID   uuid.NullUUID
	getAllChirpsDescAuthorID  uuid.NullUUID
	getChirpByIDID            uuid.UUID
	deleteChirpByIDID         uuid.UUID
	getUserByEmailEmail       string
	createRefreshTokenParams  database.CreateRefreshTokenParams
	getRefreshTokenByIDToken  string
	revokeRefreshTokenToken   string
	createUserParams          database.CreateUserParams
	updateUserParams          database.UpdateUserParams
	updateChirpyRedFlagParams database.UpdateChirpyRedFlagParams
	updateChirpyRedFlagCalls  int
	deleteAllUsersCalls       int
}

func (m *mockQuerier) CreateChirp(ctx context.Context, arg database.CreateChirpParams) (database.Chirp, error) {
	m.createChirpParams = arg
	if m.createChirpFn == nil {
		return database.Chirp{}, errUnstubbed
	}
	return m.createChirpFn(ctx, arg)
}

func (m *mockQuerier) GetAllChirpsAsc(ctx context.Context, authorID uuid.NullUUID) ([]database.Chirp, error) {
	m.getAllChirpsAscAuthorID = authorID
	if m.getAllChirpsAscFn == nil {
		return nil, errUnstubbed
	}
	return m.getAllChirpsAscFn(ctx, authorID)
}

func (m *mockQuerier) GetAllChirpsDesc(ctx context.Context, authorID uuid.NullUUID) ([]database.Chirp, error) {
	m.getAllChirpsDescAuthorID = authorID
	if m.getAllChirpsDescFn == nil {
		return nil, errUnstubbed
	}
	return m.getAllChirpsDescFn(ctx, authorID)
}

func (m *mockQuerier) GetChirpByID(ctx context.Context, id uuid.UUID) (database.Chirp, error) {
	m.getChirpByIDID = id
	if m.getChirpByIDFn == nil {
		return database.Chirp{}, errUnstubbed
	}
	return m.getChirpByIDFn(ctx, id)
}

func (m *mockQuerier) DeleteChirpByID(ctx context.Context, id uuid.UUID) error {
	m.deleteChirpByIDID = id
	if m.deleteChirpByIDFn == nil {
		return errUnstubbed
	}
	return m.deleteChirpByIDFn(ctx, id)
}

func (m *mockQuerier) GetUserByEmail(ctx context.Context, email string) (database.User, error) {
	m.getUserByEmailEmail = email
	if m.getUserByEmailFn == nil {
		return database.User{}, errUnstubbed
	}
	return m.getUserByEmailFn(ctx, email)
}

func (m *mockQuerier) CreateRefreshToken(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error) {
	m.createRefreshTokenParams = arg
	if m.createRefreshTokenFn == nil {
		return database.RefreshToken{}, errUnstubbed
	}
	return m.createRefreshTokenFn(ctx, arg)
}

func (m *mockQuerier) GetRefreshTokenByID(ctx context.Context, token string) (database.RefreshToken, error) {
	m.getRefreshTokenByIDToken = token
	if m.getRefreshTokenByIDFn == nil {
		return database.RefreshToken{}, errUnstubbed
	}
	return m.getRefreshTokenByIDFn(ctx, token)
}

func (m *mockQuerier) RevokeRefreshToken(ctx context.Context, token string) error {
	m.revokeRefreshTokenToken = token
	if m.revokeRefreshTokenFn == nil {
		return errUnstubbed
	}
	return m.revokeRefreshTokenFn(ctx, token)
}

func (m *mockQuerier) CreateUser(ctx context.Context, arg database.CreateUserParams) (database.User, error) {
	m.createUserParams = arg
	if m.createUserFn == nil {
		return database.User{}, errUnstubbed
	}
	return m.createUserFn(ctx, arg)
}

func (m *mockQuerier) UpdateUser(ctx context.Context, arg database.UpdateUserParams) (database.User, error) {
	m.updateUserParams = arg
	if m.updateUserFn == nil {
		return database.User{}, errUnstubbed
	}
	return m.updateUserFn(ctx, arg)
}

func (m *mockQuerier) UpdateChirpyRedFlag(ctx context.Context, arg database.UpdateChirpyRedFlagParams) (database.User, error) {
	m.updateChirpyRedFlagCalls++
	m.updateChirpyRedFlagParams = arg
	if m.updateChirpyRedFlagFn == nil {
		return database.User{}, errUnstubbed
	}
	return m.updateChirpyRedFlagFn(ctx, arg)
}

func (m *mockQuerier) DeleteAllUsers(ctx context.Context) error {
	m.deleteAllUsersCalls++
	if m.deleteAllUsersFn == nil {
		return errUnstubbed
	}
	return m.deleteAllUsersFn(ctx)
}
