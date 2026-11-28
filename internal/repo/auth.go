package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"navbox/internal/model"
)

type AuthRepo interface {
	WithAuthTx(ctx context.Context, fn func(AuthRepo) error) error
	ListAccounts(ctx context.Context) ([]model.AdminAccount, error)
	GetAccount(ctx context.Context, id uuid.UUID) (*model.AdminAccount, error)
	GetAccountByUsername(ctx context.Context, username string) (*model.AdminAccount, error)
	CreateAccount(ctx context.Context, account *model.AdminAccount) error
	UpdateAccount(ctx context.Context, account *model.AdminAccount) error
	BindLegacySessions(ctx context.Context, accountID uuid.UUID) error
	ListActiveSessions(ctx context.Context, accountID uuid.UUID, now time.Time) ([]model.AdminSession, error)
	GetSession(ctx context.Context, id uuid.UUID, now time.Time) (*model.AdminSession, error)
	TouchSession(ctx context.Context, tokenHash string, now time.Time) (bool, error)
	DeleteAccountSessions(ctx context.Context, accountID uuid.UUID, keepHash string) error
	GetAdminSetting(ctx context.Context) (*model.AdminSetting, error)
	SaveAdminSetting(ctx context.Context, setting *model.AdminSetting) error
	UpdateAdminPasswordHash(ctx context.Context, passwordHash string) error
	ResetAdminPassword(ctx context.Context, setting *model.AdminSetting) error
	CreateSession(ctx context.Context, session *model.AdminSession) error
	GetSessionByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*model.AdminSession, error)
	UpdateSessionExpiresAt(ctx context.Context, tokenHash string, now time.Time, expiresAt time.Time) (bool, error)
	DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error
	DeleteSessionsExcept(ctx context.Context, tokenHash string) error
	DeleteAllSessions(ctx context.Context) error
	DeleteExpiredSessions(ctx context.Context, now time.Time) error
}

type authRepo struct {
	db *gorm.DB
}

func NewAuthRepo(db *gorm.DB) AuthRepo {
	return &authRepo{db: db}
}

func (r *authRepo) WithAuthTx(ctx context.Context, fn func(AuthRepo) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 通过单例行锁串行化账号变更、登录及迁移，防止最后一个管理员被并发降级。
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.AdminSetting{ID: 1}).Error; err != nil {
			return fmt.Errorf("ensure auth lock row: %w", err)
		}
		var setting model.AdminSetting
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Take(&setting, "id = ?", 1).Error; err != nil {
			return fmt.Errorf("lock auth settings: %w", err)
		}
		return fn(&authRepo{db: tx})
	})
}

func (r *authRepo) ListAccounts(ctx context.Context) ([]model.AdminAccount, error) {
	var accounts []model.AdminAccount
	if err := r.db.WithContext(ctx).Order("created_at ASC, id ASC").Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("list admin accounts: %w", err)
	}
	return accounts, nil
}

func (r *authRepo) GetAccount(ctx context.Context, id uuid.UUID) (*model.AdminAccount, error) {
	var account model.AdminAccount
	if err := r.db.WithContext(ctx).Take(&account, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get admin account: %w", err)
	}
	return &account, nil
}

func (r *authRepo) GetAccountByUsername(ctx context.Context, username string) (*model.AdminAccount, error) {
	var account model.AdminAccount
	if err := r.db.WithContext(ctx).Where("username = ?", username).Take(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get admin account by username: %w", err)
	}
	return &account, nil
}

func (r *authRepo) CreateAccount(ctx context.Context, account *model.AdminAccount) error {
	if err := r.db.WithContext(ctx).Create(account).Error; err != nil {
		return fmt.Errorf("create admin account: %w", err)
	}
	return nil
}

func (r *authRepo) UpdateAccount(ctx context.Context, account *model.AdminAccount) error {
	if err := r.db.WithContext(ctx).Model(account).Updates(map[string]any{
		"username":      account.Username,
		"password_hash": account.PasswordHash,
		"role":          account.Role,
		"disabled":      account.Disabled,
		"last_login_at": account.LastLoginAt,
	}).Error; err != nil {
		return fmt.Errorf("update admin account: %w", err)
	}
	return nil
}

func (r *authRepo) BindLegacySessions(ctx context.Context, accountID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Model(&model.AdminSession{}).Where("account_id IS NULL").Updates(map[string]any{
		"account_id":   accountID,
		"last_seen_at": gorm.Expr("COALESCE(last_seen_at, updated_at, created_at)"),
	}).Error; err != nil {
		return fmt.Errorf("bind legacy admin sessions: %w", err)
	}
	return nil
}

func (r *authRepo) ListActiveSessions(ctx context.Context, accountID uuid.UUID, now time.Time) ([]model.AdminSession, error) {
	var sessions []model.AdminSession
	if err := r.db.WithContext(ctx).Where("account_id = ? AND expires_at > ?", accountID, now).
		Order("created_at DESC, id DESC").Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("list active admin sessions: %w", err)
	}
	return sessions, nil
}

func (r *authRepo) GetSession(ctx context.Context, id uuid.UUID, now time.Time) (*model.AdminSession, error) {
	var session model.AdminSession
	if err := r.db.WithContext(ctx).Where("id = ? AND expires_at > ?", id, now).Take(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get admin session by id: %w", err)
	}
	return &session, nil
}

func (r *authRepo) TouchSession(ctx context.Context, tokenHash string, now time.Time) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.AdminSession{}).
		Where("token_hash = ? AND expires_at > ?", tokenHash, now).Update("last_seen_at", now)
	if result.Error != nil {
		return false, fmt.Errorf("touch admin session: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *authRepo) DeleteAccountSessions(ctx context.Context, accountID uuid.UUID, keepHash string) error {
	if err := r.db.WithContext(ctx).Where("account_id = ? AND token_hash <> ?", accountID, keepHash).
		Delete(&model.AdminSession{}).Error; err != nil {
		return fmt.Errorf("delete account sessions: %w", err)
	}
	return nil
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

func (r *authRepo) ResetAdminPassword(ctx context.Context, setting *model.AdminSetting) error {
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).Create(setting).Error; err != nil {
			return fmt.Errorf("save admin setting: %w", err)
		}
		if err := tx.Where("1 = 1").Delete(&model.AdminSession{}).Error; err != nil {
			return fmt.Errorf("delete all admin sessions: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("reset admin password: %w", err)
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

func (r *authRepo) UpdateSessionExpiresAt(ctx context.Context, tokenHash string, now time.Time, expiresAt time.Time) (bool, error) {
	result := r.db.WithContext(ctx).
		Model(&model.AdminSession{}).
		Where("token_hash = ? AND expires_at > ?", tokenHash, now).
		Update("expires_at", expiresAt)
	if result.Error != nil {
		return false, fmt.Errorf("update admin session expires at: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
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

func (r *authRepo) DeleteAllSessions(ctx context.Context) error {
	if err := r.db.WithContext(ctx).
		Where("1 = 1").
		Delete(&model.AdminSession{}).Error; err != nil {
		return fmt.Errorf("delete all admin sessions: %w", err)
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
