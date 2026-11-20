package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"navbox/internal/model"
)

type AuditFilter struct {
	ActorType string
	Action    string
	Status    string
	Query     string
	From      *time.Time
	To        *time.Time
	Limit     int
	Offset    int
}

type AuditRepo interface {
	Create(ctx context.Context, log *model.AuditLog) error
	List(ctx context.Context, filter AuditFilter) ([]model.AuditLog, int64, error)
	ExportAll(ctx context.Context, filter AuditFilter) ([]model.AuditLog, error)
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type auditRepo struct {
	db *gorm.DB
}

func NewAuditRepo(db *gorm.DB) AuditRepo {
	return &auditRepo{db: db}
}

func (r *auditRepo) Create(ctx context.Context, log *model.AuditLog) error {
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}
	if strings.TrimSpace(log.DetailJSON) == "" {
		log.DetailJSON = "{}"
	}
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

func (r *auditRepo) List(ctx context.Context, filter AuditFilter) ([]model.AuditLog, int64, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	baseQuery := applyAuditFilter(r.db.WithContext(ctx).Model(&model.AuditLog{}), filter)

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	var logs []model.AuditLog
	if err := baseQuery.Order("id DESC").Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	return logs, total, nil
}

func (r *auditRepo) ExportAll(ctx context.Context, filter AuditFilter) ([]model.AuditLog, error) {
	const pageSize = 500
	var all []model.AuditLog
	for offset := 0; ; offset += pageSize {
		f := filter
		f.Limit = pageSize
		f.Offset = offset
		batch, _, err := r.List(ctx, f)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < pageSize {
			break
		}
	}
	return all, nil
}

func (r *auditRepo) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&model.AuditLog{})
	if result.Error != nil {
		return 0, fmt.Errorf("delete audit logs before %s: %w", cutoff.Format(time.RFC3339), result.Error)
	}
	return result.RowsAffected, nil
}

func applyAuditFilter(query *gorm.DB, filter AuditFilter) *gorm.DB {
	if v := strings.TrimSpace(filter.ActorType); v != "" {
		query = query.Where("actor_type = ?", v)
	}
	if v := strings.TrimSpace(filter.Action); v != "" {
		query = query.Where("action ILIKE ?", "%"+v+"%")
	}
	if v := strings.TrimSpace(filter.Status); v != "" {
		query = query.Where("status = ?", v)
	}
	if v := strings.TrimSpace(filter.Query); v != "" {
		pattern := "%" + v + "%"
		query = query.Where(
			"actor_name ILIKE ? OR resource_type ILIKE ? OR resource_id ILIKE ? OR detail_json ILIKE ? OR ip_address ILIKE ?",
			pattern, pattern, pattern, pattern, pattern,
		)
	}
	if filter.From != nil {
		query = query.Where("created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("created_at <= ?", *filter.To)
	}
	return query
}
