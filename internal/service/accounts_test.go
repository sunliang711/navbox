package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"navbox/internal/dto"
	"navbox/internal/model"
)

func accountTestService(t *testing.T) (AuthService, *fakeAuthRepo, LoginResult) {
	t.Helper()
	hash, err := hashPassword("admin-password")
	require.NoError(t, err)
	r := newFakeAuthRepo()
	r.setting = &model.AdminSetting{ID: 1, PasswordHash: hash, Initialized: true}
	s := NewAuthService(AuthServiceParams{Lifecycle: fakeLifecycle{}, Config: testAuthConfig(), Logger: zerolog.Nop(), Repo: r})
	require.NoError(t, s.EnsureInitialized(context.Background()))
	login, err := s.Login(context.Background(), "admin", "admin-password", "192.0.2.1", "test-browser")
	require.NoError(t, err)
	return s, r, login
}

func createTestAccount(t *testing.T, s AuthService, token, username, role string) dto.AccountResp {
	t.Helper()
	account, err := s.CreateAccount(context.Background(), token, dto.CreateAccountReq{
		Username: username, Role: role, Password: "viewer-password", CurrentPassword: "admin-password",
	})
	require.NoError(t, err)
	return account
}

func TestAccountMigrationKeepsLegacySessionAndPrimaryIdentity(t *testing.T) {
	ctx := context.Background()
	hash, err := hashPassword("legacy-password")
	require.NoError(t, err)
	r := newFakeAuthRepo()
	r.setting = &model.AdminSetting{ID: 1, PasswordHash: hash, Initialized: true}
	r.sessions[hashToken("legacy-token")] = &model.AdminSession{
		Base:      model.Base{ID: uuid.New(), CreatedAt: time.Now().Add(-time.Hour)},
		TokenHash: hashToken("legacy-token"), ExpiresAt: time.Now().Add(time.Hour),
	}
	s := NewAuthService(AuthServiceParams{Lifecycle: fakeLifecycle{}, Config: testAuthConfig(), Logger: zerolog.Nop(), Repo: r})
	require.NoError(t, s.EnsureInitialized(ctx))
	require.NoError(t, s.ValidateSession(ctx, "legacy-token"))
	list, err := s.ListSessions(ctx, "legacy-token")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.True(t, list[0].Current)
	require.False(t, list[0].LastSeenAt.IsZero())
	account, err := r.GetAccountByUsername(ctx, "admin")
	require.NoError(t, err)
	require.Equal(t, hash, account.PasswordHash)
	name := "root-admin"
	_, err = s.UpdateAccount(ctx, "legacy-token", account.ID, dto.UpdateAccountReq{CurrentPassword: "legacy-password", Username: &name})
	require.NoError(t, err)
	require.NoError(t, s.EnsureInitialized(ctx))
	require.Len(t, r.accounts, 1)
	require.NoError(t, s.ResetPassword(ctx, "restored-password"))
	_, err = s.Login(ctx, name, "restored-password", "", "")
	require.NoError(t, err)
	require.ErrorIs(t, s.ValidateSession(ctx, "legacy-token"), ErrUnauthenticated)
}

func TestAccountPasswordAndSessionIsolation(t *testing.T) {
	ctx := context.Background()
	s, r, admin := accountTestService(t)
	createTestAccount(t, s, admin.Token, "viewer-one", model.RoleViewer)
	viewer, err := s.Login(ctx, "viewer-one", "viewer-password", "192.0.2.2", "another-browser")
	require.NoError(t, err)
	viewerOther, err := s.Login(ctx, "viewer-one", "viewer-password", "192.0.2.3", "mobile-browser")
	require.NoError(t, err)
	expiredAt := time.Now().Add(-time.Hour)
	expired := &model.AdminSession{Base: model.Base{ID: uuid.New()}, AccountID: r.sessions[hashToken(viewer.Token)].AccountID, TokenHash: "expired-hash", ExpiresAt: expiredAt}
	require.NoError(t, r.CreateSession(ctx, expired))
	sessions, err := s.ListSessions(ctx, viewer.Token)
	require.NoError(t, err)
	require.Len(t, sessions, 2)
	current := 0
	for _, session := range sessions {
		if session.Current {
			current++
		}
		require.NotEqual(t, "192.0.2.1", session.ClientIP)
	}
	require.Equal(t, 1, current)
	encoded, err := json.Marshal(sessions)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "token_hash")
	require.NotContains(t, string(encoded), hashToken(viewer.Token))
	adminSession := r.sessions[hashToken(admin.Token)]
	_, err = s.RevokeSession(ctx, viewer.Token, adminSession.ID)
	require.ErrorIs(t, err, ErrNotFound)
	_, err = s.ListAccounts(ctx, viewer.Token)
	require.ErrorIs(t, err, ErrForbidden)
	_, err = s.CreateAccount(ctx, viewer.Token, dto.CreateAccountReq{CurrentPassword: "viewer-password", Username: "forbidden", Password: "long-password", Role: model.RoleAdmin})
	require.ErrorIs(t, err, ErrForbidden)
	require.NoError(t, s.ChangePassword(ctx, viewer.Token, "viewer-password", "new-viewer-password"))
	require.NoError(t, s.ValidateSession(ctx, viewer.Token))
	require.ErrorIs(t, s.ValidateSession(ctx, viewerOther.Token), ErrUnauthenticated)
	require.NoError(t, s.ValidateSession(ctx, admin.Token))
	currentSession := r.sessions[hashToken(viewer.Token)]
	isCurrent, err := s.RevokeSession(ctx, viewer.Token, currentSession.ID)
	require.NoError(t, err)
	require.True(t, isCurrent)
	require.ErrorIs(t, s.ValidateSession(ctx, viewer.Token), ErrUnauthenticated)
}

func TestAccountChangesRevokeTargetSessions(t *testing.T) {
	for _, kind := range []string{"role", "disable", "password"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			s, _, admin := accountTestService(t)
			account := createTestAccount(t, s, admin.Token, "target-user", model.RoleViewer)
			login, err := s.Login(ctx, account.Username, "viewer-password", "", "")
			require.NoError(t, err)
			req := dto.UpdateAccountReq{CurrentPassword: "admin-password"}
			role, disabled, password := model.RoleAdmin, true, "reset-password"
			switch kind {
			case "role":
				req.Role = &role
			case "disable":
				req.Disabled = &disabled
			case "password":
				req.NewPassword = &password
			}
			_, err = s.UpdateAccount(ctx, admin.Token, uuid.MustParse(account.ID), req)
			require.NoError(t, err)
			require.ErrorIs(t, s.ValidateSession(ctx, login.Token), ErrUnauthenticated)
			require.NoError(t, s.ValidateSession(ctx, admin.Token))
			if kind == "disable" {
				_, err = s.Login(ctx, account.Username, "viewer-password", "", "")
				require.ErrorIs(t, err, ErrInvalidCredentials)
				disabled = false
				_, err = s.UpdateAccount(ctx, admin.Token, uuid.MustParse(account.ID), dto.UpdateAccountReq{CurrentPassword: "admin-password", Disabled: &disabled})
				require.NoError(t, err)
				_, err = s.Login(ctx, account.Username, "viewer-password", "", "")
				require.NoError(t, err)
			}
		})
	}
}

func TestAccountManagementGuards(t *testing.T) {
	ctx := context.Background()
	s, r, admin := accountTestService(t)
	id := uuid.MustParse(admin.User.ID)
	viewer, disabled := model.RoleViewer, true
	_, err := s.UpdateAccount(ctx, admin.Token, id, dto.UpdateAccountReq{CurrentPassword: "wrong-password", Role: &viewer})
	require.ErrorIs(t, err, ErrInvalidCredentials)
	_, err = s.UpdateAccount(ctx, admin.Token, id, dto.UpdateAccountReq{CurrentPassword: "admin-password", Role: &viewer})
	require.ErrorIs(t, err, ErrLastAdmin)
	require.Equal(t, model.RoleAdmin, r.accounts[id].Role)
	_, err = s.UpdateAccount(ctx, admin.Token, id, dto.UpdateAccountReq{CurrentPassword: "admin-password", Disabled: &disabled})
	require.ErrorIs(t, err, ErrInvalidInput)
	createTestAccount(t, s, admin.Token, "second-admin", model.RoleAdmin)
	_, err = s.CreateAccount(ctx, admin.Token, dto.CreateAccountReq{Username: "second-admin", Role: model.RoleViewer, Password: "some-password", CurrentPassword: "admin-password"})
	require.ErrorIs(t, err, ErrUsernameTaken)
	_, err = s.UpdateAccount(ctx, admin.Token, uuid.New(), dto.UpdateAccountReq{CurrentPassword: "admin-password", Role: &viewer})
	require.ErrorIs(t, err, ErrNotFound)
	_, err = s.UpdateAccount(ctx, admin.Token, id, dto.UpdateAccountReq{CurrentPassword: "admin-password", Role: &viewer})
	require.NoError(t, err)
	require.ErrorIs(t, s.ValidateSession(ctx, admin.Token), ErrUnauthenticated)
}

func TestAccountInputValidation(t *testing.T) {
	s, _, admin := accountTestService(t)
	for _, tt := range []struct {
		name, username, password, role string
		want                           error
	}{
		{"invalid username", "bad/name", "valid-password", model.RoleViewer, ErrInvalidInput},
		{"invalid role", "new-user", "valid-password", "owner", ErrInvalidInput},
		{"short password", "new-user", "short", model.RoleViewer, ErrWeakPassword},
		{"blank password", "new-user", "        ", model.RoleViewer, ErrWeakPassword},
		{"bcrypt byte limit", "new-user", strings.Repeat("密", 25), model.RoleViewer, ErrWeakPassword},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.CreateAccount(context.Background(), admin.Token, dto.CreateAccountReq{Username: tt.username, Password: tt.password, Role: tt.role, CurrentPassword: "admin-password"})
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestSessionActivityIsThrottled(t *testing.T) {
	s, r, admin := accountTestService(t)
	session := r.sessions[hashToken(admin.Token)]
	old := time.Now().Add(-2 * time.Minute)
	session.LastSeenAt = &old
	_, err := s.ValidateSessionWithRenewal(context.Background(), admin.Token)
	require.NoError(t, err)
	require.True(t, session.LastSeenAt.After(old))
	seen := *session.LastSeenAt
	_, err = s.ValidateSessionWithRenewal(context.Background(), admin.Token)
	require.NoError(t, err)
	require.Equal(t, seen, *session.LastSeenAt)
}

func TestAdminResetOwnPasswordKeepsCurrentSession(t *testing.T) {
	s, _, admin := accountTestService(t)
	ctx := context.Background()
	other, err := s.Login(ctx, "admin", "admin-password", "192.0.2.2", "other-device")
	require.NoError(t, err)
	password := "reset-own-password"
	_, err = s.UpdateAccount(ctx, admin.Token, uuid.MustParse(admin.User.ID), dto.UpdateAccountReq{CurrentPassword: "admin-password", NewPassword: &password})
	require.NoError(t, err)
	require.NoError(t, s.ValidateSession(ctx, admin.Token))
	require.ErrorIs(t, s.ValidateSession(ctx, other.Token), ErrUnauthenticated)
	_, err = s.Login(ctx, "admin", password, "", "")
	require.NoError(t, err)
}
