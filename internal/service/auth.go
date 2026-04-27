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
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"golang.org/x/crypto/bcrypt"

	"navbox/internal/config"
	"navbox/internal/model"
	"navbox/internal/repo"
)

const (
	adminSettingID        = 1
	sessionTokenBytes     = 32
	minAdminPasswordBytes = 8
	passwordAlphabet      = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%^&*"
)

var (
	ErrUnauthenticated    = errors.New("unauthenticated")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrWeakPassword       = errors.New("weak password")
)

type AuthService interface {
	EnsureInitialized(ctx context.Context) error
	Login(ctx context.Context, password string) (LoginResult, error)
	ValidateSession(ctx context.Context, token string) error
	Logout(ctx context.Context, token string) error
	ChangePassword(ctx context.Context, token string, currentPassword string, newPassword string) error
}

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
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
	setting, err := s.repo.GetAdminSetting(ctx)
	if err != nil {
		return err
	}
	if setting != nil && setting.Initialized && setting.PasswordHash != "" {
		return nil
	}

	password, err := generatePassword(s.cfg.Auth.InitialPasswordLength)
	if err != nil {
		return err
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		return err
	}

	if err := s.repo.SaveAdminSetting(ctx, &model.AdminSetting{
		ID:           adminSettingID,
		PasswordHash: passwordHash,
		Initialized:  true,
	}); err != nil {
		return err
	}

	s.logger.Warn().Str("password", password).Msg("Admin initial password generated")
	return nil
}

func (s *authService) Login(ctx context.Context, password string) (LoginResult, error) {
	setting, err := s.repo.GetAdminSetting(ctx)
	if err != nil {
		return LoginResult{}, err
	}
	if setting == nil || !setting.Initialized || setting.PasswordHash == "" {
		return LoginResult{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(setting.PasswordHash), []byte(password)); err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	token, err := generateSessionToken()
	if err != nil {
		return LoginResult{}, err
	}
	sessionTTL, err := time.ParseDuration(s.cfg.Auth.SessionTTL)
	if err != nil {
		return LoginResult{}, fmt.Errorf("parse session ttl: %w", err)
	}
	now := time.Now().UTC()
	expiresAt := now.Add(sessionTTL)

	if err := s.repo.DeleteExpiredSessions(ctx, now); err != nil {
		return LoginResult{}, err
	}
	if err := s.repo.CreateSession(ctx, &model.AdminSession{
		TokenHash: hashToken(token),
		ExpiresAt: expiresAt,
	}); err != nil {
		return LoginResult{}, err
	}

	return LoginResult{Token: token, ExpiresAt: expiresAt}, nil
}

func (s *authService) ValidateSession(ctx context.Context, token string) error {
	if token == "" {
		return ErrUnauthenticated
	}
	session, err := s.repo.GetSessionByTokenHash(ctx, hashToken(token), time.Now().UTC())
	if err != nil {
		return err
	}
	if session == nil {
		return ErrUnauthenticated
	}
	return nil
}

func (s *authService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.repo.DeleteSessionByTokenHash(ctx, hashToken(token))
}

func (s *authService) ChangePassword(ctx context.Context, token string, currentPassword string, newPassword string) error {
	if len(newPassword) < minAdminPasswordBytes {
		return ErrWeakPassword
	}
	if err := s.ValidateSession(ctx, token); err != nil {
		return err
	}

	setting, err := s.repo.GetAdminSetting(ctx)
	if err != nil {
		return err
	}
	if setting == nil || !setting.Initialized || setting.PasswordHash == "" {
		return ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(setting.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrInvalidCredentials
	}

	passwordHash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	tokenHash := hashToken(token)
	if err := s.repo.UpdateAdminPasswordHash(ctx, passwordHash); err != nil {
		return err
	}
	if err := s.repo.DeleteSessionsExcept(ctx, tokenHash); err != nil {
		return err
	}
	return nil
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
