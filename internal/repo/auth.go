package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"navbox/internal/model"
)

type AuthRepo interface {
	GetAdminSetting(ctx context.Context) (*model.AdminSetting, error)
	SaveAdminSetting(ctx context.Context, setting *model.AdminSetting) error
	UpdateAdminPasswordHash(ctx context.Context, passwordHash string) error
	CreateSession(ctx context.Context, session *model.AdminSession) error
	GetSessionByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*model.AdminSession, error)
	DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error
	DeleteSessionsExcept(ctx context.Context, tokenHash string) error
	DeleteExpiredSessions(ctx context.Context, now time.Time) error
}

type authRepo struct {
	db *gorm.DB
}

func NewAuthRepo(db *gorm.DB) AuthRepo {
	return &authRepo{db: db}
}

func (r *authRepo) GetAdminSetting(ctx context.Context) (*model.AdminSetting, error) {
	var setting model.AdminSetting
	if err := r.db.WithContext(ctx).Take(&setting, "id = ?", 1).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get admin setting: %w", err)
	}
	return &setting, nil
}

func (r *authRepo) SaveAdminSetting(ctx context.Context, setting *model.AdminSetting) error {
	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).
		Create(setting).Error; err != nil {
		return fmt.Errorf("save admin setting: %w", err)
	}
	return nil
}

func (r *authRepo) UpdateAdminPasswordHash(ctx context.Context, passwordHash string) error {
	result := r.db.WithContext(ctx).
		Model(&model.AdminSetting{}).
		Where("id = ?", 1).
		Updates(map[string]any{
			"password_hash": passwordHash,
			"initialized":   true,
		})
	if result.Error != nil {
		return fmt.Errorf("update admin password hash: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *authRepo) CreateSession(ctx context.Context, session *model.AdminSession) error {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("create admin session: %w", err)
	}
	return nil
}

func (r *authRepo) GetSessionByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*model.AdminSession, error) {
	var session model.AdminSession
	if err := r.db.WithContext(ctx).
		Where("token_hash = ? AND expires_at > ?", tokenHash, now).
		Take(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get admin session: %w", err)
	}
	return &session, nil
}

func (r *authRepo) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	if err := r.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		Delete(&model.AdminSession{}).Error; err != nil {
		return fmt.Errorf("delete admin session: %w", err)
	}
	return nil
}

func (r *authRepo) DeleteSessionsExcept(ctx context.Context, tokenHash string) error {
	if err := r.db.WithContext(ctx).
		Where("token_hash <> ?", tokenHash).
		Delete(&model.AdminSession{}).Error; err != nil {
		return fmt.Errorf("delete other admin sessions: %w", err)
	}
	return nil
}

func (r *authRepo) DeleteExpiredSessions(ctx context.Context, now time.Time) error {
	if err := r.db.WithContext(ctx).
		Where("expires_at <= ?", now).
		Delete(&model.AdminSession{}).Error; err != nil {
		return fmt.Errorf("delete expired admin sessions: %w", err)
	}
	return nil
}
