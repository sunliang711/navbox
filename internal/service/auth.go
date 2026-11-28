package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"golang.org/x/crypto/bcrypt"

	"navbox/internal/config"
	"navbox/internal/dto"
	"navbox/internal/model"
	"navbox/internal/repo"
)

const (
	adminSettingID                 = 1
	sessionTokenBytes              = 32
	sessionRenewalThresholdDivisor = 2
	sessionActivityInterval        = time.Minute
	minAdminPasswordBytes          = 8
	passwordAlphabet               = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%^&*"
)

var (
	ErrUnauthenticated    = errors.New("unauthenticated")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrWeakPassword       = errors.New("weak password")
)

type AuthService interface {
	EnsureInitialized(ctx context.Context) error
	Login(ctx context.Context, username string, password string, clientIP string, userAgent string) (LoginResult, error)
	ValidateSession(ctx context.Context, token string) error
	ValidateSessionWithRenewal(ctx context.Context, token string) (SessionValidationResult, error)
	Logout(ctx context.Context, token string) error
	ChangePassword(ctx context.Context, token string, currentPassword string, newPassword string) error
	ResetPassword(ctx context.Context, newPassword string) error
	ListSessions(ctx context.Context, token string) ([]dto.ActiveSessionResp, error)
	RevokeSession(ctx context.Context, token string, id uuid.UUID) (bool, error)
	ListAccounts(ctx context.Context, token string) ([]dto.AccountResp, error)
	CreateAccount(ctx context.Context, token string, req dto.CreateAccountReq) (dto.AccountResp, error)
	UpdateAccount(ctx context.Context, token string, id uuid.UUID, req dto.UpdateAccountReq) (dto.AccountResp, error)
}

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	User      dto.AccountResp
}

type SessionValidationResult struct {
	ExpiresAt time.Time
	Renewed   bool
	User      dto.AccountResp
}

type AuthServiceParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Config    config.Config
	Logger    zerolog.Logger
	Repo      repo.AuthRepo
}

type authService struct {
	cfg    config.Config
	logger zerolog.Logger
	repo   repo.AuthRepo
}

func NewAuthService(params AuthServiceParams) AuthService {
	svc := &authService{
		cfg:    params.Config,
		logger: params.Logger,
		repo:   params.Repo,
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return svc.EnsureInitialized(ctx)
		},
	})

	return svc
}

func (s *authService) EnsureInitialized(ctx context.Context) error {
	var initialPassword string
	err := s.repo.WithAuthTx(ctx, func(tx repo.AuthRepo) error {
		accounts, err := tx.ListAccounts(ctx)
		if err != nil {
			return err
		}
		if len(accounts) > 0 {
			return nil
		}
		setting, err := tx.GetAdminSetting(ctx)
		if err != nil {
			return err
		}
		if setting == nil || !setting.Initialized || setting.PasswordHash == "" {
			initialPassword, err = generatePassword(s.cfg.Auth.InitialPasswordLength)
			if err != nil {
				return err
			}
			passwordHash, err := hashPassword(initialPassword)
			if err != nil {
				return err
			}
			setting = &model.AdminSetting{ID: adminSettingID, PasswordHash: passwordHash, Initialized: true}
			if err := tx.SaveAdminSetting(ctx, setting); err != nil {
				return err
			}
		}
		// 默认账号沿用原密码，并接管已有会话，避免升级时强制退出。
		account := &model.AdminAccount{
			Base: model.Base{ID: uuid.New()}, Username: "admin", PasswordHash: setting.PasswordHash,
			Role: model.RoleAdmin, IsPrimary: true,
		}
		if err := tx.CreateAccount(ctx, account); err != nil {
			return err
		}
		return tx.BindLegacySessions(ctx, account.ID)
	})
	if err == nil && initialPassword != "" {
		s.logger.Warn().Str("password", initialPassword).Msg("Admin initial password generated")
	}
	return err
}

func (s *authService) Login(ctx context.Context, username string, password string, clientIP string, userAgent string) (LoginResult, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		username = "admin"
	}
	account, err := s.repo.GetAccountByUsername(ctx, username)
	if err != nil {
		return LoginResult{}, err
	}
	if account == nil || account.Disabled {
		return LoginResult{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)); err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	token, err := generateSessionToken()
	if err != nil {
		return LoginResult{}, err
	}
	sessionTTL, err := s.sessionTTL()
	if err != nil {
		return LoginResult{}, err
	}
	now := time.Now().UTC()
	expiresAt := now.Add(sessionTTL)

	err = s.repo.WithAuthTx(ctx, func(tx repo.AuthRepo) error {
		current, err := tx.GetAccount(ctx, account.ID)
		if err != nil {
			return err
		}
		// 验密后再次核对，防止并发停用、改密或改名期间创建旧凭据会话。
		if current == nil || current.Disabled || current.PasswordHash != account.PasswordHash || current.Username != username {
			return ErrInvalidCredentials
		}
		if err := tx.DeleteExpiredSessions(ctx, now); err != nil {
			return err
		}
		if err := tx.CreateSession(ctx, &model.AdminSession{
			Base: model.Base{ID: uuid.New()}, AccountID: &current.ID,
			TokenHash: hashToken(token), ExpiresAt: expiresAt,
			ClientIP: clientIP, UserAgent: truncateUserAgent(userAgent), LastSeenAt: &now,
		}); err != nil {
			return err
		}
		current.LastLoginAt = &now
		account = current
		return tx.UpdateAccount(ctx, current)
	})
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{Token: token, ExpiresAt: expiresAt, User: accountDTO(account)}, nil
}

func (s *authService) ValidateSession(ctx context.Context, token string) error {
	_, err := s.validateSession(ctx, token, false)
	return err
}

func (s *authService) ValidateSessionWithRenewal(ctx context.Context, token string) (SessionValidationResult, error) {
	return s.validateSession(ctx, token, true)
}

func (s *authService) validateSession(ctx context.Context, token string, renew bool) (SessionValidationResult, error) {
	if token == "" {
		return SessionValidationResult{}, ErrUnauthenticated
	}
	now := time.Now().UTC()
	tokenHash := hashToken(token)
	session, err := s.repo.GetSessionByTokenHash(ctx, tokenHash, now)
	if err != nil {
		return SessionValidationResult{}, err
	}
	if session == nil || session.AccountID == nil {
		return SessionValidationResult{}, ErrUnauthenticated
	}
	account, err := s.repo.GetAccount(ctx, *session.AccountID)
	if err != nil {
		return SessionValidationResult{}, err
	}
	if account == nil || account.Disabled || (account.Role != model.RoleAdmin && account.Role != model.RoleViewer) {
		return SessionValidationResult{}, ErrUnauthenticated
	}
	result := SessionValidationResult{ExpiresAt: session.ExpiresAt, User: accountDTO(account)}
	if !renew {
		return result, nil
	}
	if session.LastSeenAt == nil || now.Sub(*session.LastSeenAt) >= sessionActivityInterval {
		updated, err := s.repo.TouchSession(ctx, tokenHash, now)
		if err != nil {
			return SessionValidationResult{}, err
		}
		if !updated {
			return SessionValidationResult{}, ErrUnauthenticated
		}
	}

	sessionTTL, err := s.sessionTTL()
	if err != nil {
		return SessionValidationResult{}, err
	}
	if session.ExpiresAt.Sub(now) >= sessionTTL/sessionRenewalThresholdDivisor {
		return result, nil
	}

	expiresAt := now.Add(sessionTTL)
	updated, err := s.repo.UpdateSessionExpiresAt(ctx, tokenHash, now, expiresAt)
	if err != nil {
		return SessionValidationResult{}, err
	}
	if !updated {
		return SessionValidationResult{}, ErrUnauthenticated
	}

	result.ExpiresAt = expiresAt
	result.Renewed = true
	return result, nil
}

func (s *authService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.repo.DeleteSessionByTokenHash(ctx, hashToken(token))
}

func (s *authService) ChangePassword(ctx context.Context, token string, currentPassword string, newPassword string) error {
	if !validAccountPassword(newPassword) {
		return ErrWeakPassword
	}
	passwordHash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.repo.WithAuthTx(ctx, func(tx repo.AuthRepo) error {
		account, _, err := s.withRepo(tx).principal(ctx, token)
		if err != nil {
			return err
		}
		if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(currentPassword)); err != nil {
			return ErrInvalidCredentials
		}
		account.PasswordHash = passwordHash
		if err := tx.UpdateAccount(ctx, account); err != nil {
			return err
		}
		return tx.DeleteAccountSessions(ctx, account.ID, hashToken(token))
	})
}

func (s *authService) ResetPassword(ctx context.Context, newPassword string) error {
	if !validAccountPassword(newPassword) {
		return ErrWeakPassword
	}

	passwordHash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.repo.WithAuthTx(ctx, func(tx repo.AuthRepo) error {
		accounts, err := tx.ListAccounts(ctx)
		if err != nil {
			return err
		}
		for _, account := range accounts {
			if !account.IsPrimary {
				continue
			}
			// 恢复入口始终找迁移时的主账号，改名后仍可恢复管理权限。
			account.PasswordHash, account.Role, account.Disabled = passwordHash, model.RoleAdmin, false
			if err := tx.UpdateAccount(ctx, &account); err != nil {
				return err
			}
			return tx.DeleteAllSessions(ctx)
		}
		return ErrNotFound
	})
}

func (s *authService) sessionTTL() (time.Duration, error) {
	ttl, err := time.ParseDuration(s.cfg.Auth.SessionTTL)
	if err != nil {
		return 0, fmt.Errorf("parse session ttl: %w", err)
	}
	return ttl, nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash admin password: %w", err)
	}
	return string(hash), nil
}

func generatePassword(length int) (string, error) {
	result := make([]byte, length)
	max := big.NewInt(int64(len(passwordAlphabet)))
	for i := range result {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("generate admin password: %w", err)
		}
		result[i] = passwordAlphabet[idx.Int64()]
	}
	return string(result), nil
}

func generateSessionToken() (string, error) {
	buf := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
