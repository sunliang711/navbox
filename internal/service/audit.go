package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog"

	"navbox/internal/dto"
	"navbox/internal/model"
	"navbox/internal/repo"
)

const (
	AuditActorAdmin  = "admin"
	AuditActorSystem = "system"

	AuditStatusSuccess = "success"
	AuditStatusFailure = "failure"
)

type AuditEvent struct {
	ActorType    string
	ActorID      string
	ActorName    string
	Action       string
	ResourceType string
	ResourceID   string
	Status       string
	IPAddress    string
	UserAgent    string
	RequestID    string
	Detail       any
}

type AuditService interface {
	Record(ctx context.Context, ev AuditEvent)
	List(ctx context.Context, query dto.AuditListQuery) (dto.AuditListResp, error)
	ExportAll(ctx context.Context, query dto.AuditListQuery) ([]dto.AuditLogResp, error)
	CleanupBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type auditService struct {
	repo   repo.AuditRepo
	logger zerolog.Logger
}

func NewAuditService(repo repo.AuditRepo, logger zerolog.Logger) AuditService {
	return &auditService{
		repo:   repo,
		logger: logger.With().Str("component", "audit").Logger(),
	}
}

func (s *auditService) Record(ctx context.Context, ev AuditEvent) {
	if s == nil || s.repo == nil {
		return
	}
	detailJSON := "{}"
	if ev.Detail != nil {
		if b, err := json.Marshal(ev.Detail); err == nil {
			detailJSON = string(b)
		}
	}
	status := ev.Status
	if status == "" {
		status = AuditStatusSuccess
	}
	actorType := ev.ActorType
	if actorType == "" {
		actorType = AuditActorAdmin
	}
	actorName := ev.ActorName
	if actorName == "" {
		actorName = actorType
	}
	actorID := ev.ActorID
	if actorID == "" {
		actorID = actorName
	}

	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()

	if err := s.repo.Create(writeCtx, &model.AuditLog{
		ActorType:    actorType,
		ActorID:      actorID,
		ActorName:    actorName,
		Action:       ev.Action,
		ResourceType: ev.ResourceType,
		ResourceID:   ev.ResourceID,
		Status:       status,
		IPAddress:    ev.IPAddress,
		UserAgent:    ev.UserAgent,
		RequestID:    ev.RequestID,
		DetailJSON:   detailJSON,
		CreatedAt:    time.Now().UTC(),
	}); err != nil {
		s.logger.Error().Err(err).Str("action", ev.Action).Msg("Failed to record audit log")
	}
}

func (s *auditService) List(ctx context.Context, query dto.AuditListQuery) (dto.AuditListResp, error) {
	logs, total, err := s.repo.List(ctx, toRepoAuditFilter(query))
	if err != nil {
		return dto.AuditListResp{}, err
	}
	items := make([]dto.AuditLogResp, 0, len(logs))
	for _, item := range logs {
		items = append(items, mapAuditLogResp(item))
	}
	return dto.AuditListResp{
		Items: items,
		Total: total,
	}, nil
}

func (s *auditService) ExportAll(ctx context.Context, query dto.AuditListQuery) ([]dto.AuditLogResp, error) {
	logs, err := s.repo.ExportAll(ctx, toRepoAuditFilter(query))
	if err != nil {
		return nil, err
	}
	items := make([]dto.AuditLogResp, 0, len(logs))
	for _, item := range logs {
		items = append(items, mapAuditLogResp(item))
	}
	return items, nil
}

func (s *auditService) CleanupBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.repo.DeleteBefore(ctx, cutoff)
}

func toRepoAuditFilter(query dto.AuditListQuery) repo.AuditFilter {
	return repo.AuditFilter{
		ActorType: query.ActorType,
		Action:    query.Action,
		Status:    query.Status,
		Query:     query.Query,
		From:      query.From,
		To:        query.To,
		Limit:     query.Limit,
		Offset:    query.Offset,
	}
}

func mapAuditLogResp(item model.AuditLog) dto.AuditLogResp {
	return dto.AuditLogResp{
		ID:           item.ID,
		ActorType:    item.ActorType,
		ActorID:      item.ActorID,
		ActorName:    item.ActorName,
		Action:       item.Action,
		ResourceType: item.ResourceType,
		ResourceID:   item.ResourceID,
		Status:       item.Status,
		IPAddress:    item.IPAddress,
		UserAgent:    item.UserAgent,
		RequestID:    item.RequestID,
		DetailJSON:   item.DetailJSON,
		CreatedAt:    item.CreatedAt,
	}
}
