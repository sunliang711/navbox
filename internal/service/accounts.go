package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"navbox/internal/dto"
	"navbox/internal/model"
	"navbox/internal/repo"
)

const (
	maxAccountPasswordBytes  = 72 // BCrypt 的密码长度上限。
	maxSessionUserAgentRunes = 512
)

var (
	ErrForbidden           = errors.New("forbidden")
	ErrLastAdmin           = errors.New("last active administrator")
	ErrUsernameTaken       = errors.New("username taken")
	accountUsernamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{2,63}$`)
)

func accountDTO(account *model.AdminAccount) dto.AccountResp {
	return dto.AccountResp{
		ID: account.ID.String(), Username: account.Username, Role: account.Role,
		Disabled: account.Disabled, LastLoginAt: account.LastLoginAt, CreatedAt: account.CreatedAt,
	}
}

func validAccountPassword(password string) bool {
	return len(password) >= minAdminPasswordBytes && len(password) <= maxAccountPasswordBytes && strings.TrimSpace(password) != ""
}

func truncateUserAgent(value string) string {
	if utf8.RuneCountInString(value) <= maxSessionUserAgentRunes {
		return value
	}
	return string([]rune(value)[:maxSessionUserAgentRunes])
}

func (s *authService) withRepo(r repo.AuthRepo) *authService {
	return &authService{cfg: s.cfg, logger: s.logger, repo: r}
}

func (s *authService) principal(ctx context.Context, token string) (*model.AdminAccount, *model.AdminSession, error) {
	if token == "" {
		return nil, nil, ErrUnauthenticated
	}
	session, err := s.repo.GetSessionByTokenHash(ctx, hashToken(token), time.Now().UTC())
	if err != nil {
		return nil, nil, err
	}
	if session == nil || session.AccountID == nil {
		return nil, nil, ErrUnauthenticated
	}
	account, err := s.repo.GetAccount(ctx, *session.AccountID)
	if err != nil {
		return nil, nil, err
	}
	if account == nil || account.Disabled || (account.Role != model.RoleAdmin && account.Role != model.RoleViewer) {
		return nil, nil, ErrUnauthenticated
	}
	return account, session, nil
}

func (s *authService) authorizeAccountChange(ctx context.Context, token, password string) (*model.AdminAccount, error) {
	account, _, err := s.principal(ctx, token)
	if err != nil {
		return nil, err
	}
	if account.Role != model.RoleAdmin {
		return nil, ErrForbidden
	}
	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return account, nil
}

func (s *authService) ListAccounts(ctx context.Context, token string) ([]dto.AccountResp, error) {
	actor, _, err := s.principal(ctx, token)
	if err != nil {
		return nil, err
	}
	if actor.Role != model.RoleAdmin {
		return nil, ErrForbidden
	}
	accounts, err := s.repo.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.AccountResp, 0, len(accounts))
	for i := range accounts {
		out = append(out, accountDTO(&accounts[i]))
	}
	return out, nil
}

func (s *authService) CreateAccount(ctx context.Context, token string, req dto.CreateAccountReq) (dto.AccountResp, error) {
	if !accountUsernamePattern.MatchString(req.Username) {
		return dto.AccountResp{}, newValidationError("用户名需为 3～64 位字母、数字、点、下划线或连字符，且以字母或数字开头")
	}
	if req.Role != model.RoleAdmin && req.Role != model.RoleViewer {
		return dto.AccountResp{}, newValidationError("角色只能是 admin 或 viewer")
	}
	if !validAccountPassword(req.Password) {
		return dto.AccountResp{}, ErrWeakPassword
	}
	hash, err := hashPassword(req.Password)
	if err != nil {
		return dto.AccountResp{}, err
	}
	account := &model.AdminAccount{Base: model.Base{ID: uuid.New()}, Username: req.Username, PasswordHash: hash, Role: req.Role}
	err = s.repo.WithAuthTx(ctx, func(tx repo.AuthRepo) error {
		if _, err := s.withRepo(tx).authorizeAccountChange(ctx, token, req.CurrentPassword); err != nil {
			return err
		}
		other, err := tx.GetAccountByUsername(ctx, req.Username)
		if err != nil {
			return err
		}
		if other != nil {
			return ErrUsernameTaken
		}
		return tx.CreateAccount(ctx, account)
	})
	return accountDTO(account), err
}

func (s *authService) UpdateAccount(ctx context.Context, token string, id uuid.UUID, req dto.UpdateAccountReq) (dto.AccountResp, error) {
	if req.Username == nil && req.Role == nil && req.Disabled == nil && req.NewPassword == nil {
		return dto.AccountResp{}, newValidationError("请提供要修改的账号信息")
	}
	if req.Username != nil && !accountUsernamePattern.MatchString(*req.Username) {
		return dto.AccountResp{}, newValidationError("用户名格式无效")
	}
	if req.Role != nil && *req.Role != model.RoleAdmin && *req.Role != model.RoleViewer {
		return dto.AccountResp{}, newValidationError("角色只能是 admin 或 viewer")
	}
	var passwordHash string
	if req.NewPassword != nil {
		if !validAccountPassword(*req.NewPassword) {
			return dto.AccountResp{}, ErrWeakPassword
		}
		var err error
		passwordHash, err = hashPassword(*req.NewPassword)
		if err != nil {
			return dto.AccountResp{}, err
		}
	}
	var out dto.AccountResp
	err := s.repo.WithAuthTx(ctx, func(tx repo.AuthRepo) error {
		actor, err := s.withRepo(tx).authorizeAccountChange(ctx, token, req.CurrentPassword)
		if err != nil {
			return err
		}
		account, err := tx.GetAccount(ctx, id)
		if err != nil {
			return err
		}
		if account == nil {
			return ErrNotFound
		}
		if req.Disabled != nil && *req.Disabled && actor.ID == id {
			return newValidationError("不能停用当前登录账号")
		}
		wasActiveAdmin := account.Role == model.RoleAdmin && !account.Disabled
		revoke := req.NewPassword != nil
		if req.Username != nil {
			other, err := tx.GetAccountByUsername(ctx, *req.Username)
			if err != nil {
				return err
			}
			if other != nil && other.ID != id {
				return ErrUsernameTaken
			}
			account.Username = *req.Username
		}
		if req.Role != nil {
			revoke = revoke || account.Role != *req.Role
			account.Role = *req.Role
		}
		if req.Disabled != nil {
			revoke = revoke || *req.Disabled
			account.Disabled = *req.Disabled
		}
		if wasActiveAdmin && (account.Role != model.RoleAdmin || account.Disabled) {
			accounts, err := tx.ListAccounts(ctx)
			if err != nil {
				return err
			}
			activeAdmins := 0
			for _, other := range accounts {
				if other.ID != id && other.Role == model.RoleAdmin && !other.Disabled {
					activeAdmins++
				}
			}
			if activeAdmins == 0 {
				return ErrLastAdmin
			}
		}
		if req.NewPassword != nil {
			account.PasswordHash = passwordHash
		}
		if err := tx.UpdateAccount(ctx, account); err != nil {
			return err
		}
		if revoke {
			keepHash := ""
			// 自己重置密码保留当前会话；角色改变时所有旧权限会话必须注销。
			if actor.ID == id && req.NewPassword != nil && req.Role == nil && req.Disabled == nil {
				keepHash = hashToken(token)
			}
			if err := tx.DeleteAccountSessions(ctx, id, keepHash); err != nil {
				return err
			}
		}
		out = accountDTO(account)
		return nil
	})
	return out, err
}

func (s *authService) ListSessions(ctx context.Context, token string) ([]dto.ActiveSessionResp, error) {
	account, current, err := s.principal(ctx, token)
	if err != nil {
		return nil, err
	}
	sessions, err := s.repo.ListActiveSessions(ctx, account.ID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	out := make([]dto.ActiveSessionResp, 0, len(sessions))
	for _, session := range sessions {
		lastSeen := session.CreatedAt
		if session.LastSeenAt != nil {
			lastSeen = *session.LastSeenAt
		}
		out = append(out, dto.ActiveSessionResp{
			ID: session.ID.String(), ClientIP: session.ClientIP, UserAgent: session.UserAgent,
			CreatedAt: session.CreatedAt, LastSeenAt: lastSeen, ExpiresAt: session.ExpiresAt, Current: session.ID == current.ID,
		})
	}
	return out, nil
}

func (s *authService) RevokeSession(ctx context.Context, token string, id uuid.UUID) (bool, error) {
	current := false
	err := s.repo.WithAuthTx(ctx, func(tx repo.AuthRepo) error {
		account, caller, err := s.withRepo(tx).principal(ctx, token)
		if err != nil {
			return err
		}
		target, err := tx.GetSession(ctx, id, time.Now().UTC())
		if err != nil {
			return err
		}
		if target == nil || target.AccountID == nil || (*target.AccountID != account.ID && account.Role != model.RoleAdmin) {
			return ErrNotFound
		}
		current = caller.ID == id
		return tx.DeleteSessionByTokenHash(ctx, target.TokenHash)
	})
	return current, err
}
