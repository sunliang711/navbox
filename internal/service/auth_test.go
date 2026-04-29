package service

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"navbox/internal/config"
	"navbox/internal/model"
)

type fakeAuthRepo struct {
	setting         *model.AdminSetting
	sessions        map[string]*model.AdminSession
	sessionRenewals int
}

type fakeLifecycle struct{}

func (fakeLifecycle) Append(fx.Hook) {}

func newFakeAuthRepo() *fakeAuthRepo {
	return &fakeAuthRepo{sessions: make(map[string]*model.AdminSession)}
}

func (r *fakeAuthRepo) GetAdminSetting(ctx context.Context) (*model.AdminSetting, error) {
	return r.setting, nil
}

func (r *fakeAuthRepo) SaveAdminSetting(ctx context.Context, setting *model.AdminSetting) error {
	r.setting = setting
	return nil
}

func (r *fakeAuthRepo) UpdateAdminPasswordHash(ctx context.Context, passwordHash string) error {
	r.setting.PasswordHash = passwordHash
	r.setting.Initialized = true
	return nil
}

func (r *fakeAuthRepo) ResetAdminPassword(ctx context.Context, setting *model.AdminSetting) error {
	r.setting = setting
	return r.DeleteAllSessions(ctx)
}

func (r *fakeAuthRepo) CreateSession(ctx context.Context, session *model.AdminSession) error {
	r.sessions[session.TokenHash] = session
	return nil
}

func (r *fakeAuthRepo) GetSessionByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*model.AdminSession, error) {
	session := r.sessions[tokenHash]
	if session == nil || !session.ExpiresAt.After(now) {
		return nil, nil
	}
	return session, nil
}

func (r *fakeAuthRepo) UpdateSessionExpiresAt(ctx context.Context, tokenHash string, now time.Time, expiresAt time.Time) (bool, error) {
	session := r.sessions[tokenHash]
	if session == nil || !session.ExpiresAt.After(now) {
		return false, nil
	}
	session.ExpiresAt = expiresAt
	r.sessionRenewals++
	return true, nil
}

func (r *fakeAuthRepo) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	delete(r.sessions, tokenHash)
	return nil
}

func (r *fakeAuthRepo) DeleteSessionsExcept(ctx context.Context, tokenHash string) error {
	for key := range r.sessions {
		if key != tokenHash {
			delete(r.sessions, key)
		}
	}
	return nil
}

func (r *fakeAuthRepo) DeleteAllSessions(ctx context.Context) error {
	for key := range r.sessions {
		delete(r.sessions, key)
	}
	return nil
}

func (r *fakeAuthRepo) DeleteExpiredSessions(ctx context.Context, now time.Time) error {
	for key, session := range r.sessions {
		if !session.ExpiresAt.After(now) {
			delete(r.sessions, key)
		}
	}
	return nil
}

func testAuthConfig() config.Config {
	return config.Config{
		Auth: config.AuthConfig{
			SessionTTL:            "24h",
			CookieName:            "navbox_admin_session",
			InitialPasswordLength: 16,
		},
	}
}

func TestAuthServiceEnsureInitialized(t *testing.T) {
	repo := newFakeAuthRepo()
	svc := NewAuthService(AuthServiceParams{
		Lifecycle: fakeLifecycle{},
		Config:    testAuthConfig(),
		Logger:    zerolog.Nop(),
		Repo:      repo,
	})

	require.NoError(t, svc.EnsureInitialized(context.Background()))
	require.NotNil(t, repo.setting)
	require.True(t, repo.setting.Initialized)
	require.NotEmpty(t, repo.setting.PasswordHash)
}

func TestAuthServiceLoginAndValidateSession(t *testing.T) {
	passwordHash, err := hashPassword("correct-password")
	require.NoError(t, err)

	repo := newFakeAuthRepo()
	repo.setting = &model.AdminSetting{
		ID:           adminSettingID,
		PasswordHash: passwordHash,
		Initialized:  true,
	}
	svc := NewAuthService(AuthServiceParams{
		Lifecycle: fakeLifecycle{},
		Config:    testAuthConfig(),
		Logger:    zerolog.Nop(),
		Repo:      repo,
	})

	result, err := svc.Login(context.Background(), "correct-password")

	require.NoError(t, err)
	require.NotEmpty(t, result.Token)
	require.NoError(t, svc.ValidateSession(context.Background(), result.Token))
	require.ErrorIs(t, svc.ValidateSession(context.Background(), "bad-token"), ErrUnauthenticated)
}

func TestAuthServiceValidateSessionWithRenewal(t *testing.T) {
	tests := []struct {
		name        string
		expiresIn   time.Duration
		wantRenewed bool
	}{
		{name: "renews when remaining ttl below half", expiresIn: 30 * time.Minute, wantRenewed: true},
		{name: "keeps session when remaining ttl enough", expiresIn: 90 * time.Minute, wantRenewed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passwordHash, err := hashPassword("correct-password")
			require.NoError(t, err)

			repo := newFakeAuthRepo()
			repo.setting = &model.AdminSetting{
				ID:           adminSettingID,
				PasswordHash: passwordHash,
				Initialized:  true,
			}
			cfg := testAuthConfig()
			cfg.Auth.SessionTTL = "2h"
			svc := NewAuthService(AuthServiceParams{
				Lifecycle: fakeLifecycle{},
				Config:    cfg,
				Logger:    zerolog.Nop(),
				Repo:      repo,
			})

			loginResult, err := svc.Login(context.Background(), "correct-password")
			require.NoError(t, err)

			session := repo.sessions[hashToken(loginResult.Token)]
			require.NotNil(t, session)
			session.ExpiresAt = time.Now().UTC().Add(tt.expiresIn)
			originalExpiresAt := session.ExpiresAt

			result, err := svc.ValidateSessionWithRenewal(context.Background(), loginResult.Token)

			require.NoError(t, err)
			require.Equal(t, tt.wantRenewed, result.Renewed)
			if tt.wantRenewed {
				require.Equal(t, 1, repo.sessionRenewals)
				require.True(t, result.ExpiresAt.After(originalExpiresAt))
				require.Equal(t, result.ExpiresAt, session.ExpiresAt)
				return
			}
			require.Equal(t, 0, repo.sessionRenewals)
			require.Equal(t, originalExpiresAt, result.ExpiresAt)
			require.Equal(t, originalExpiresAt, session.ExpiresAt)
		})
	}
}

func TestAuthServiceChangePassword(t *testing.T) {
	passwordHash, err := hashPassword("old-password")
	require.NoError(t, err)

	repo := newFakeAuthRepo()
	repo.setting = &model.AdminSetting{
		ID:           adminSettingID,
		PasswordHash: passwordHash,
		Initialized:  true,
	}
	svc := NewAuthService(AuthServiceParams{
		Lifecycle: fakeLifecycle{},
		Config:    testAuthConfig(),
		Logger:    zerolog.Nop(),
		Repo:      repo,
	})

	result, err := svc.Login(context.Background(), "old-password")
	require.NoError(t, err)

	require.NoError(t, svc.ChangePassword(context.Background(), result.Token, "old-password", "new-password"))

	_, err = svc.Login(context.Background(), "old-password")
	require.ErrorIs(t, err, ErrInvalidCredentials)

	_, err = svc.Login(context.Background(), "new-password")
	require.NoError(t, err)
}

func TestAuthServiceResetPassword(t *testing.T) {
	passwordHash, err := hashPassword("old-password")
	require.NoError(t, err)

	repo := newFakeAuthRepo()
	repo.setting = &model.AdminSetting{
		ID:           adminSettingID,
		PasswordHash: passwordHash,
		Initialized:  true,
	}
	svc := NewAuthService(AuthServiceParams{
		Lifecycle: fakeLifecycle{},
		Config:    testAuthConfig(),
		Logger:    zerolog.Nop(),
		Repo:      repo,
	})

	result, err := svc.Login(context.Background(), "old-password")
	require.NoError(t, err)
	require.NotEmpty(t, repo.sessions)

	require.NoError(t, svc.ResetPassword(context.Background(), "new-password"))
	require.Empty(t, repo.sessions)

	_, err = svc.Login(context.Background(), "old-password")
	require.ErrorIs(t, err, ErrInvalidCredentials)

	_, err = svc.Login(context.Background(), "new-password")
	require.NoError(t, err)
	require.ErrorIs(t, svc.ValidateSession(context.Background(), result.Token), ErrUnauthenticated)
}

func TestAuthServiceResetPasswordRejectsWeakPassword(t *testing.T) {
	svc := NewAuthService(AuthServiceParams{
		Lifecycle: fakeLifecycle{},
		Config:    testAuthConfig(),
		Logger:    zerolog.Nop(),
		Repo:      newFakeAuthRepo(),
	})

	err := svc.ResetPassword(context.Background(), "short")
	require.ErrorIs(t, err, ErrWeakPassword)
}
