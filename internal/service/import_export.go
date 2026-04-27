package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"navbox/internal/dto"
	"navbox/internal/model"
	"navbox/internal/repo"
	"navbox/internal/storage"
)

const (
	archiveVersion        = 1
	archiveJSONName       = "navbox.json"
	archiveIconsDir       = "icons"
	maxArchiveJSONBytes   = 8 * 1024 * 1024
	MaxImportArchiveBytes = 64 * 1024 * 1024
)

type ImportExportService interface {
	Export(ctx context.Context, req dto.ExportReq) (*dto.ExportArchiveResp, error)
	Import(ctx context.Context, file multipart.File) (*dto.ImportReportResp, error)
}

type importExportService struct {
	repo  repo.ImportExportRepo
	store *storage.IconStore
}

func NewImportExportService(repo repo.ImportExportRepo, store *storage.IconStore) ImportExportService {
	return &importExportService{repo: repo, store: store}
}

func (s *importExportService) Export(ctx context.Context, req dto.ExportReq) (*dto.ExportArchiveResp, error) {
	siteIDs, err := parseUUIDs(req.SiteIDs)
	if err != nil {
		return nil, err
	}
	tagIDs, err := parseUUIDs(req.TagIDs)
	if err != nil {
		return nil, err
	}

	filter := repo.ExportFilter{SiteIDs: siteIDs, TagIDs: tagIDs}
	data, err := s.repo.LoadExportData(ctx, filter)
	if err != nil {
		return nil, err
	}

	var icons []model.Icon
	if len(siteIDs) == 0 && len(tagIDs) == 0 {
		icons, err = s.repo.ListAllIcons(ctx)
	} else {
		icons, err = s.repo.ListIconsByFileNames(ctx, collectArchiveIconFileNames(data.Sites, data.Tags))
	}
	if err != nil {
		return nil, err
	}

	content, err := s.buildArchive(data, icons)
	if err != nil {
		return nil, err
	}

	return &dto.ExportArchiveResp{
		FileName: "navbox-export-" + time.Now().UTC().Format("20060102150405") + ".zip",
		Content:  content,
	}, nil
}

func (s *importExportService) Import(ctx context.Context, file multipart.File) (*dto.ImportReportResp, error) {
	content, err := io.ReadAll(io.LimitReader(file, MaxImportArchiveBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read import archive: %w", err)
	}
	if int64(len(content)) > MaxImportArchiveBytes {
		return nil, ErrInvalidInput
	}

	archive, iconFiles, err := s.readArchive(content)
	if err != nil {
		return nil, err
	}
	batch, filesToWrite, report, err := s.buildImportBatch(ctx, archive, iconFiles)
	if err != nil {
		return nil, err
	}

	writtenFiles, err := s.writeImportIconFiles(filesToWrite)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Import(ctx, batch); err != nil {
		removeFiles(writtenFiles)
		return nil, err
	}

	return &report, nil
}

func (s *importExportService) buildArchive(data repo.ExportData, icons []model.Icon) ([]byte, error) {
	archive := dto.NavboxArchive{
		Version:    archiveVersion,
		ExportedAt: time.Now().UTC(),
		Sites:      mapArchiveSites(data.Sites, data.SiteTags),
		Tags:       mapArchiveTags(data.Tags),
		Icons:      mapArchiveIcons(icons),
	}

	jsonContent, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode archive json: %w", err)
	}

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	if err := addZipFile(zipWriter, archiveJSONName, jsonContent); err != nil {
		_ = zipWriter.Close()
		return nil, err
	}

	addedIcons := make(map[string]struct{}, len(icons))
	for _, icon := range icons {
		if _, ok := addedIcons[icon.FileName]; ok {
			continue
		}
		addedIcons[icon.FileName] = struct{}{}

		iconPath, err := s.store.Path(icon.FileName)
		if err != nil {
			_ = zipWriter.Close()
			return nil, ErrInvalidInput
		}
		content, err := os.ReadFile(iconPath)
		if err != nil {
			_ = zipWriter.Close()
			return nil, fmt.Errorf("read icon file: %w", err)
		}
		if err := addZipFile(zipWriter, archiveIconsDir+"/"+icon.FileName, content); err != nil {
			_ = zipWriter.Close()
			return nil, err
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("close export archive: %w", err)
	}
	return buf.Bytes(), nil
}

func (s *importExportService) readArchive(content []byte) (dto.NavboxArchive, map[string][]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return dto.NavboxArchive{}, nil, ErrInvalidInput
	}

	var archive dto.NavboxArchive
	hasJSON := false
	iconFiles := make(map[string][]byte)
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		name := file.Name
		cleanName := path.Clean(name)
		if name != cleanName || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "../") || strings.Contains(name, "\\") {
			return dto.NavboxArchive{}, nil, ErrInvalidInput
		}

		switch {
		case name == archiveJSONName:
			if hasJSON {
				return dto.NavboxArchive{}, nil, ErrInvalidInput
			}
			jsonContent, err := readZipFile(file, maxArchiveJSONBytes)
			if err != nil {
				return dto.NavboxArchive{}, nil, err
			}
			if err := json.Unmarshal(jsonContent, &archive); err != nil {
				return dto.NavboxArchive{}, nil, ErrInvalidInput
			}
			hasJSON = true
		case strings.HasPrefix(name, archiveIconsDir+"/"):
			fileName := path.Base(name)
			if fileName == "" || name != archiveIconsDir+"/"+fileName || fileName != filepath.Base(fileName) {
				return dto.NavboxArchive{}, nil, ErrInvalidInput
			}
			iconContent, err := readZipFile(file, s.store.MaxBytes())
			if err != nil {
				return dto.NavboxArchive{}, nil, err
			}
			iconFiles[fileName] = iconContent
		default:
			return dto.NavboxArchive{}, nil, ErrInvalidInput
		}
	}
	if !hasJSON || archive.Version != archiveVersion {
		return dto.NavboxArchive{}, nil, ErrInvalidInput
	}
	return archive, iconFiles, nil
}

func (s *importExportService) buildImportBatch(ctx context.Context, archive dto.NavboxArchive, iconFiles map[string][]byte) (repo.ImportBatch, map[string][]byte, dto.ImportReportResp, error) {
	sites, relations, siteIDs, err := parseArchiveSites(archive.Sites)
	if err != nil {
		return repo.ImportBatch{}, nil, dto.ImportReportResp{}, err
	}
	tags, tagIDs, tagNames, err := parseArchiveTags(archive.Tags)
	if err != nil {
		return repo.ImportBatch{}, nil, dto.ImportReportResp{}, err
	}
	icons, iconFileData, iconIDs, iconSHA256, err := s.parseArchiveIcons(archive.Icons, iconFiles)
	if err != nil {
		return repo.ImportBatch{}, nil, dto.ImportReportResp{}, err
	}

	existing, err := s.repo.LoadImportExistingData(ctx, repo.ImportLookupKeys{
		SiteIDs:    siteIDs,
		TagIDs:     tagIDs,
		TagNames:   tagNames,
		IconIDs:    iconIDs,
		IconSHA256: iconSHA256,
	})
	if err != nil {
		return repo.ImportBatch{}, nil, dto.ImportReportResp{}, err
	}

	report := dto.ImportReportResp{}
	tagIDMap := make(map[uuid.UUID]uuid.UUID, len(tags))
	importedSiteIDs := make(map[uuid.UUID]struct{}, len(sites))
	batch := repo.ImportBatch{}
	filesToWrite := make(map[string][]byte)

	existingSiteIDs := mapExistingSites(existing.Sites)
	existingTagsByID, existingTagsByName := mapExistingTags(existing.Tags)
	existingIconsByID, existingIconsBySHA256 := mapExistingIcons(existing.Icons)

	for _, tag := range tags {
		if existingTag, ok := existingTagsByID[tag.ID]; ok {
			tagIDMap[tag.ID] = existingTag.ID
			report.Skipped.Tags++
			addImportConflict(&report, "tag", tag.ID.String(), tag.Name, "Tag ID 已存在")
			continue
		}
		if existingTag, ok := existingTagsByName[tag.Name]; ok {
			tagIDMap[tag.ID] = existingTag.ID
			report.Skipped.Tags++
			addImportConflict(&report, "tag", tag.ID.String(), tag.Name, "Tag 名称已存在")
			continue
		}
		tagIDMap[tag.ID] = tag.ID
		batch.Tags = append(batch.Tags, tag)
		report.Imported.Tags++
	}

	for _, icon := range icons {
		if _, ok := existingIconsByID[icon.ID]; ok {
			report.Skipped.Icons++
			addImportConflict(&report, "icon", icon.ID.String(), icon.FileName, "icon ID 已存在")
			continue
		}
		if _, ok := existingIconsBySHA256[icon.SHA256]; ok {
			report.Skipped.Icons++
			addImportConflict(&report, "icon", icon.ID.String(), icon.FileName, "icon 内容已存在")
			continue
		}
		batch.Icons = append(batch.Icons, icon)
		filesToWrite[icon.FileName] = iconFileData[icon.FileName]
		report.Imported.Icons++
	}

	for _, site := range sites {
		if _, ok := existingSiteIDs[site.ID]; ok {
			report.Skipped.Sites++
			addImportConflict(&report, "site", site.ID.String(), site.Title, "网站 ID 已存在")
			continue
		}
		importedSiteIDs[site.ID] = struct{}{}
		batch.Sites = append(batch.Sites, site)
		report.Imported.Sites++
	}

	relationSet := make(map[string]struct{}, len(relations))
	for _, relation := range relations {
		if _, ok := importedSiteIDs[relation.SiteID]; !ok {
			report.Skipped.Relations++
			continue
		}
		targetTagID, ok := tagIDMap[relation.TagID]
		if !ok {
			report.Skipped.Relations++
			addImportConflict(&report, "relation", relation.SiteID.String(), relation.TagID.String(), "关联 Tag 不存在")
			continue
		}
		key := relation.SiteID.String() + ":" + targetTagID.String()
		if _, ok := relationSet[key]; ok {
			continue
		}
		relationSet[key] = struct{}{}
		batch.SiteTags = append(batch.SiteTags, model.SiteTag{
			SiteID:    relation.SiteID,
			TagID:     targetTagID,
			CreatedAt: time.Now().UTC(),
		})
		report.Imported.Relations++
	}

	return batch, filesToWrite, report, nil
}

func (s *importExportService) parseArchiveIcons(items []dto.ArchiveIcon, iconFiles map[string][]byte) ([]model.Icon, map[string][]byte, []uuid.UUID, []string, error) {
	icons := make([]model.Icon, 0, len(items))
	iconFileData := make(map[string][]byte, len(items))
	ids := make([]uuid.UUID, 0, len(items))
	hashes := make([]string, 0, len(items))
	seenIDs := make(map[uuid.UUID]struct{}, len(items))
	seenHashes := make(map[string]struct{}, len(items))
	seenFileNames := make(map[string]struct{}, len(items))

	for _, item := range items {
		id, err := parseUUID(item.ID)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		fileName := strings.TrimSpace(item.FileName)
		if fileName == "" || fileName != filepath.Base(fileName) {
			return nil, nil, nil, nil, ErrInvalidInput
		}
		shaValue := strings.ToLower(strings.TrimSpace(item.SHA256))
		if len(shaValue) != 64 || item.SizeBytes <= 0 {
			return nil, nil, nil, nil, ErrInvalidInput
		}
		if _, ok := seenIDs[id]; ok {
			return nil, nil, nil, nil, ErrInvalidInput
		}
		if _, ok := seenHashes[shaValue]; ok {
			return nil, nil, nil, nil, ErrInvalidInput
		}
		if _, ok := seenFileNames[fileName]; ok {
			return nil, nil, nil, nil, ErrInvalidInput
		}

		content, ok := iconFiles[fileName]
		if !ok {
			return nil, nil, nil, nil, ErrInvalidInput
		}
		if err := s.validateArchiveIcon(item, content); err != nil {
			return nil, nil, nil, nil, err
		}

		seenIDs[id] = struct{}{}
		seenHashes[shaValue] = struct{}{}
		seenFileNames[fileName] = struct{}{}
		ids = append(ids, id)
		hashes = append(hashes, shaValue)
		iconFileData[fileName] = content
		icons = append(icons, model.Icon{
			Base: model.Base{
				ID:        id,
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
			},
			FileName:  fileName,
			FilePath:  fileName,
			SHA256:    shaValue,
			MIMEType:  strings.TrimSpace(item.MIMEType),
			SizeBytes: item.SizeBytes,
		})
	}

	return icons, iconFileData, ids, hashes, nil
}

func (s *importExportService) validateArchiveIcon(item dto.ArchiveIcon, content []byte) error {
	if len(content) == 0 || int64(len(content)) > s.store.MaxBytes() || int64(len(content)) != item.SizeBytes {
		return ErrInvalidInput
	}
	sum := sha256.Sum256(content)
	shaValue := strings.ToLower(strings.TrimSpace(item.SHA256))
	if hex.EncodeToString(sum[:]) != shaValue {
		return ErrInvalidInput
	}
	mimeType := http.DetectContentType(content)
	if mimeType != strings.TrimSpace(item.MIMEType) {
		return ErrInvalidInput
	}
	ext, ok := storage.AllowedIconExtension(mimeType)
	if !ok {
		return ErrInvalidInput
	}
	if item.FileName != shaValue+ext {
		return ErrInvalidInput
	}
	return nil
}

func (s *importExportService) writeImportIconFiles(files map[string][]byte) ([]string, error) {
	writtenFiles := make([]string, 0, len(files))
	for fileName, content := range files {
		targetPath, err := s.store.Path(fileName)
		if err != nil {
			removeFiles(writtenFiles)
			return nil, ErrInvalidInput
		}
		if _, err := os.Stat(targetPath); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			removeFiles(writtenFiles)
			return nil, fmt.Errorf("stat import icon file: %w", err)
		}
		if err := os.WriteFile(targetPath, content, 0640); err != nil {
			removeFiles(writtenFiles)
			return nil, fmt.Errorf("write import icon file: %w", err)
		}
		writtenFiles = append(writtenFiles, targetPath)
	}
	return writtenFiles, nil
}

func parseArchiveSites(items []dto.ArchiveSite) ([]model.Site, []model.SiteTag, []uuid.UUID, error) {
	sites := make([]model.Site, 0, len(items))
	relations := make([]model.SiteTag, 0)
	ids := make([]uuid.UUID, 0, len(items))
	seenIDs := make(map[uuid.UUID]struct{}, len(items))

	for _, item := range items {
		id, err := parseUUID(item.ID)
		if err != nil {
			return nil, nil, nil, err
		}
		if _, ok := seenIDs[id]; ok {
			return nil, nil, nil, ErrInvalidInput
		}
		if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.DefaultURL) == "" {
			return nil, nil, nil, ErrInvalidInput
		}

		seenIDs[id] = struct{}{}
		ids = append(ids, id)
		sites = append(sites, model.Site{
			Base: model.Base{
				ID:        id,
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
			},
			Title:           strings.TrimSpace(item.Title),
			Description:     strings.TrimSpace(item.Description),
			DefaultURL:      strings.TrimSpace(item.DefaultURL),
			LANURL:          strings.TrimSpace(item.LANURL),
			OpenMethod:      strings.TrimSpace(item.OpenMethod),
			IconType:        strings.TrimSpace(item.IconType),
			IconValue:       strings.TrimSpace(item.IconValue),
			BackgroundColor: strings.TrimSpace(item.BackgroundColor),
			OnlyName:        item.OnlyName,
			IsFavorite:      item.IsFavorite,
			SortOrder:       item.SortOrder,
		})

		tagSet := make(map[uuid.UUID]struct{}, len(item.TagIDs))
		for _, value := range item.TagIDs {
			tagID, err := parseUUID(value)
			if err != nil {
				return nil, nil, nil, err
			}
			if _, ok := tagSet[tagID]; ok {
				continue
			}
			tagSet[tagID] = struct{}{}
			relations = append(relations, model.SiteTag{SiteID: id, TagID: tagID})
		}
	}

	return sites, relations, ids, nil
}

func parseArchiveTags(items []dto.ArchiveTag) ([]model.Tag, []uuid.UUID, []string, error) {
	tags := make([]model.Tag, 0, len(items))
	ids := make([]uuid.UUID, 0, len(items))
	names := make([]string, 0, len(items))
	seenIDs := make(map[uuid.UUID]struct{}, len(items))
	seenNames := make(map[string]struct{}, len(items))

	for _, item := range items {
		id, err := parseUUID(item.ID)
		if err != nil {
			return nil, nil, nil, err
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			return nil, nil, nil, ErrInvalidInput
		}
		if _, ok := seenIDs[id]; ok {
			return nil, nil, nil, ErrInvalidInput
		}
		if _, ok := seenNames[name]; ok {
			return nil, nil, nil, ErrInvalidInput
		}

		seenIDs[id] = struct{}{}
		seenNames[name] = struct{}{}
		ids = append(ids, id)
		names = append(names, name)
		tags = append(tags, model.Tag{
			Base: model.Base{
				ID:        id,
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
			},
			Name:      name,
			Icon:      strings.TrimSpace(item.Icon),
			Color:     strings.TrimSpace(item.Color),
			SortOrder: item.SortOrder,
			IsDefault: item.IsDefault,
			IsEnabled: item.IsEnabled,
		})
	}

	return tags, ids, names, nil
}

func mapArchiveSites(sites []model.Site, relations []model.SiteTag) []dto.ArchiveSite {
	tagIDsBySiteID := make(map[uuid.UUID][]string, len(sites))
	for _, relation := range relations {
		tagIDsBySiteID[relation.SiteID] = append(tagIDsBySiteID[relation.SiteID], relation.TagID.String())
	}

	result := make([]dto.ArchiveSite, 0, len(sites))
	for _, site := range sites {
		result = append(result, dto.ArchiveSite{
			ID:              site.ID.String(),
			Title:           site.Title,
			Description:     site.Description,
			DefaultURL:      site.DefaultURL,
			LANURL:          site.LANURL,
			OpenMethod:      site.OpenMethod,
			IconType:        site.IconType,
			IconValue:       site.IconValue,
			BackgroundColor: site.BackgroundColor,
			OnlyName:        site.OnlyName,
			IsFavorite:      site.IsFavorite,
			SortOrder:       site.SortOrder,
			TagIDs:          tagIDsBySiteID[site.ID],
			CreatedAt:       site.CreatedAt,
			UpdatedAt:       site.UpdatedAt,
		})
	}
	return result
}

func mapArchiveTags(tags []model.Tag) []dto.ArchiveTag {
	result := make([]dto.ArchiveTag, 0, len(tags))
	for _, tag := range tags {
		result = append(result, dto.ArchiveTag{
			ID:        tag.ID.String(),
			Name:      tag.Name,
			Icon:      tag.Icon,
			Color:     tag.Color,
			SortOrder: tag.SortOrder,
			IsDefault: tag.IsDefault,
			IsEnabled: tag.IsEnabled,
			CreatedAt: tag.CreatedAt,
			UpdatedAt: tag.UpdatedAt,
		})
	}
	return result
}

func mapArchiveIcons(icons []model.Icon) []dto.ArchiveIcon {
	result := make([]dto.ArchiveIcon, 0, len(icons))
	for _, icon := range icons {
		result = append(result, dto.ArchiveIcon{
			ID:        icon.ID.String(),
			FileName:  icon.FileName,
			FilePath:  icon.FilePath,
			SHA256:    icon.SHA256,
			MIMEType:  icon.MIMEType,
			SizeBytes: icon.SizeBytes,
			CreatedAt: icon.CreatedAt,
			UpdatedAt: icon.UpdatedAt,
		})
	}
	return result
}

func collectArchiveIconFileNames(sites []model.Site, tags []model.Tag) []string {
	seen := make(map[string]struct{})
	for _, site := range sites {
		if fileName, ok := localIconFileName(site.IconValue); ok {
			seen[fileName] = struct{}{}
		}
	}
	for _, tag := range tags {
		if fileName, ok := localIconFileName(tag.Icon); ok {
			seen[fileName] = struct{}{}
		}
	}

	result := make([]string, 0, len(seen))
	for fileName := range seen {
		result = append(result, fileName)
	}
	return result
}

func localIconFileName(value string) (string, bool) {
	value = strings.TrimSpace(value)
	switch {
	case strings.HasPrefix(value, "/uploads/"):
		value = strings.TrimPrefix(value, "/uploads/")
	case strings.HasPrefix(value, "uploads/"):
		value = strings.TrimPrefix(value, "uploads/")
	}
	if value == "" || strings.Contains(value, "/") || strings.Contains(value, "\\") || value != filepath.Base(value) {
		return "", false
	}
	if !strings.Contains(value, ".") {
		return "", false
	}
	return value, true
}

func mapExistingSites(sites []model.Site) map[uuid.UUID]model.Site {
	result := make(map[uuid.UUID]model.Site, len(sites))
	for _, site := range sites {
		result[site.ID] = site
	}
	return result
}

func mapExistingTags(tags []model.Tag) (map[uuid.UUID]model.Tag, map[string]model.Tag) {
	byID := make(map[uuid.UUID]model.Tag, len(tags))
	byName := make(map[string]model.Tag, len(tags))
	for _, tag := range tags {
		byID[tag.ID] = tag
		byName[tag.Name] = tag
	}
	return byID, byName
}

func mapExistingIcons(icons []model.Icon) (map[uuid.UUID]model.Icon, map[string]model.Icon) {
	byID := make(map[uuid.UUID]model.Icon, len(icons))
	bySHA256 := make(map[string]model.Icon, len(icons))
	for _, icon := range icons {
		byID[icon.ID] = icon
		bySHA256[icon.SHA256] = icon
	}
	return byID, bySHA256
}

func addZipFile(writer *zip.Writer, name string, content []byte) error {
	fileWriter, err := writer.Create(name)
	if err != nil {
		return fmt.Errorf("create zip file: %w", err)
	}
	if _, err := fileWriter.Write(content); err != nil {
		return fmt.Errorf("write zip file: %w", err)
	}
	return nil
}

func readZipFile(file *zip.File, maxBytes int64) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, ErrInvalidInput
	}
	defer reader.Close()

	content, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read zip file: %w", err)
	}
	if int64(len(content)) > maxBytes {
		return nil, ErrInvalidInput
	}
	return content, nil
}

func addImportConflict(report *dto.ImportReportResp, itemType string, id string, name string, reason string) {
	report.Conflicts = append(report.Conflicts, dto.ImportConflictResp{
		Type:   itemType,
		ID:     id,
		Name:   name,
		Reason: reason,
	})
}

func removeFiles(paths []string) {
	for _, item := range paths {
		_ = os.Remove(item)
	}
}
