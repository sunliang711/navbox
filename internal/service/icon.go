package service

import (
	"context"
	"errors"
	"mime/multipart"

	"navbox/internal/dto"
	"navbox/internal/model"
	"navbox/internal/repo"
	"navbox/internal/storage"
)

type IconService interface {
	UploadIcon(ctx context.Context, file multipart.File) (*dto.IconResp, error)
	FetchIcon(ctx context.Context, websiteURL string) (*dto.IconResp, error)
}

type iconService struct {
	repo  repo.IconRepo
	store *storage.IconStore
}

func NewIconService(repo repo.IconRepo, store *storage.IconStore) IconService {
	return &iconService{repo: repo, store: store}
}

func (s *iconService) UploadIcon(ctx context.Context, file multipart.File) (*dto.IconResp, error) {
	stored, err := s.store.Save(file)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidIcon) || errors.Is(err, storage.ErrIconTooLarge) {
			return nil, ErrInvalidInput
		}
		return nil, err
	}
	return s.saveStoredIcon(ctx, stored)
}

func (s *iconService) saveStoredIcon(ctx context.Context, stored storage.StoredIcon) (*dto.IconResp, error) {
	existing, err := s.repo.GetBySHA256(ctx, stored.SHA256)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		resp := mapIcon(*existing)
		return &resp, nil
	}

	icon := &model.Icon{
		FileName:  stored.FileName,
		FilePath:  stored.FilePath,
		SHA256:    stored.SHA256,
		MIMEType:  stored.MIMEType,
		SizeBytes: stored.SizeBytes,
	}
	if err := s.repo.Create(ctx, icon); err != nil {
		return nil, err
	}

	resp := mapIcon(*icon)
	return &resp, nil
}

func mapIcon(icon model.Icon) dto.IconResp {
	return dto.IconResp{
		ID:        icon.ID.String(),
		FileName:  icon.FileName,
		FilePath:  icon.FilePath,
		URL:       "/uploads/" + icon.FileName,
		SHA256:    icon.SHA256,
		MIMEType:  icon.MIMEType,
		SizeBytes: icon.SizeBytes,
		CreatedAt: icon.CreatedAt,
	}
}
