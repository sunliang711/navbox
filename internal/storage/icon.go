package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	detectBytes = 512
)

var (
	ErrInvalidIcon  = errors.New("invalid icon")
	ErrIconTooLarge = errors.New("icon too large")
)

type IconStore struct {
	dir      string
	maxBytes int64
}

type StoredIcon struct {
	FileName  string
	FilePath  string
	URL       string
	SHA256    string
	MIMEType  string
	SizeBytes int64
}

func NewIconStore(dir string, maxBytes int64) (*IconStore, error) {
	if dir == "" {
		return nil, errors.New("upload dir is required")
	}
	if maxBytes <= 0 {
		return nil, errors.New("upload max bytes must be greater than zero")
	}
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &IconStore{dir: dir, maxBytes: maxBytes}, nil
}

func (s *IconStore) Save(file multipart.File) (StoredIcon, error) {
	tmp, err := os.CreateTemp(s.dir, ".icon-*")
	if err != nil {
		return StoredIcon{}, fmt.Errorf("create temp icon file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	hash := sha256.New()
	firstBytes := make([]byte, 0, detectBytes)
	buf := make([]byte, 32*1024)
	var size int64

	for {
		n, readErr := file.Read(buf)
		if n > 0 {
			size += int64(n)
			if size > s.maxBytes {
				_ = tmp.Close()
				return StoredIcon{}, ErrIconTooLarge
			}
			chunk := buf[:n]
			if len(firstBytes) < detectBytes {
				need := detectBytes - len(firstBytes)
				if len(chunk) < need {
					need = len(chunk)
				}
				firstBytes = append(firstBytes, chunk[:need]...)
			}
			if _, err := hash.Write(chunk); err != nil {
				_ = tmp.Close()
				return StoredIcon{}, fmt.Errorf("hash icon file: %w", err)
			}
			if _, err := tmp.Write(chunk); err != nil {
				_ = tmp.Close()
				return StoredIcon{}, fmt.Errorf("write temp icon file: %w", err)
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			_ = tmp.Close()
			return StoredIcon{}, fmt.Errorf("read icon file: %w", readErr)
		}
	}
	if err := tmp.Close(); err != nil {
		return StoredIcon{}, fmt.Errorf("close temp icon file: %w", err)
	}
	if size == 0 {
		return StoredIcon{}, ErrInvalidIcon
	}

	mimeType := http.DetectContentType(firstBytes)
	ext, ok := allowedIconExtension(mimeType)
	if !ok {
		return StoredIcon{}, ErrInvalidIcon
	}

	sum := hex.EncodeToString(hash.Sum(nil))
	fileName := sum + ext
	finalPath := filepath.Join(s.dir, fileName)
	if _, err := os.Stat(finalPath); err == nil {
		return StoredIcon{
			FileName:  fileName,
			FilePath:  fileName,
			URL:       "/uploads/" + fileName,
			SHA256:    sum,
			MIMEType:  mimeType,
			SizeBytes: size,
		}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return StoredIcon{}, fmt.Errorf("stat icon file: %w", err)
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		return StoredIcon{}, fmt.Errorf("save icon file: %w", err)
	}

	return StoredIcon{
		FileName:  fileName,
		FilePath:  fileName,
		URL:       "/uploads/" + fileName,
		SHA256:    sum,
		MIMEType:  mimeType,
		SizeBytes: size,
	}, nil
}

func (s *IconStore) Dir() string {
	return s.dir
}

func allowedIconExtension(mimeType string) (string, bool) {
	switch strings.ToLower(mimeType) {
	case "image/png":
		return ".png", true
	case "image/jpeg":
		return ".jpg", true
	case "image/gif":
		return ".gif", true
	case "image/webp":
		return ".webp", true
	case "image/x-icon", "image/vnd.microsoft.icon":
		return ".ico", true
	default:
		return "", false
	}
}
