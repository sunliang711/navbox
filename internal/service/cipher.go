package service

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"navbox/internal/config"
	"navbox/internal/crypto"
	"navbox/internal/repo"
)

const masterKeySettingKey = "security.master_key"

// SecretCipher 为定时任务等需要持久化凭据的功能提供加解密能力。
type SecretCipher interface {
	EncryptString(plaintext string) (string, error)
	DecryptString(encoded string) (string, error)
}

type SecretCipherParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Config    config.Config
	Logger    zerolog.Logger
	Repo      repo.SystemSettingRepo
}

type secretCipherService struct {
	cfg    config.Config
	logger zerolog.Logger
	repo   repo.SystemSettingRepo

	mu     sync.RWMutex
	cipher *crypto.Cipher
}

func NewSecretCipher(params SecretCipherParams) SecretCipher {
	svc := &secretCipherService{
		cfg:    params.Config,
		logger: params.Logger.With().Str("component", "cipher").Logger(),
		repo:   params.Repo,
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return svc.EnsureInitialized(ctx)
		},
	})

	return svc
}

// NewStaticSecretCipher 用于单元测试，使用固定 32 字节密钥构建 SecretCipher。
func NewStaticSecretCipher(masterKey []byte) (SecretCipher, error) {
	c, err := crypto.NewCipher(masterKey)
	if err != nil {
		return nil, err
	}
	return &secretCipherService{cipher: c}, nil
}

func (s *secretCipherService) EnsureInitialized(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cipher != nil {
		return nil
	}

	rawKey := strings.TrimSpace(s.cfg.Security.MasterKey)
	if rawKey != "" {
		keyBytes, err := crypto.DecodeMasterKey(rawKey)
		if err != nil {
			return fmt.Errorf("decode configured master key: %w", err)
		}
		c, err := crypto.NewCipher(keyBytes)
		if err != nil {
			return err
		}
		s.cipher = c
		s.logger.Info().Msg("Secret cipher initialized from configured master key")
		return nil
	}

	if s.repo == nil {
		return fmt.Errorf("system setting repository is required when security.master_key is empty")
	}

	existing, err := s.repo.Get(ctx, masterKeySettingKey)
	if err != nil {
		return err
	}
	if existing == nil || strings.TrimSpace(existing.ValueText) == "" {
		generated, err := crypto.GenerateMasterKeyHex()
		if err != nil {
			return err
		}
		if err := s.repo.SetIfAbsent(ctx, masterKeySettingKey, generated); err != nil {
			return err
		}
		existing, err = s.repo.Get(ctx, masterKeySettingKey)
		if err != nil {
			return err
		}
		if existing == nil || strings.TrimSpace(existing.ValueText) == "" {
			return fmt.Errorf("load persisted master key failed")
		}
		s.logger.Info().Msg("Generated and persisted default secret master key")
	}

	keyBytes, err := crypto.DecodeMasterKey(existing.ValueText)
	if err != nil {
		return fmt.Errorf("decode persisted master key: %w", err)
	}
	c, err := crypto.NewCipher(keyBytes)
	if err != nil {
		return err
	}
	s.cipher = c
	return nil
}

func (s *secretCipherService) getCipher() (*crypto.Cipher, error) {
	s.mu.RLock()
	c := s.cipher
	s.mu.RUnlock()
	if c != nil {
		return c, nil
	}
	if err := s.EnsureInitialized(context.Background()); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cipher, nil
}

func (s *secretCipherService) EncryptString(plaintext string) (string, error) {
	c, err := s.getCipher()
	if err != nil {
		return "", err
	}
	return c.EncryptString(plaintext)
}

func (s *secretCipherService) DecryptString(encoded string) (string, error) {
	c, err := s.getCipher()
	if err != nil {
		return "", err
	}
	return c.DecryptString(encoded)
}
