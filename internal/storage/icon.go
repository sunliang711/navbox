package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
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
	mimeTypeSVG = "image/svg+xml"
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
	return s.SaveReader(file)
}

func (s *IconStore) SaveReader(reader io.Reader) (StoredIcon, error) {
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
		n, readErr := reader.Read(buf)
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

	content := firstBytes
	if DetectIconContentType(firstBytes) == mimeTypeSVG {
		content, err = os.ReadFile(tmpPath)
		if err != nil {
			return StoredIcon{}, fmt.Errorf("read temp icon file: %w", err)
		}
	}
	mimeType, ext, err := ValidateIconContent(content)
	if err != nil {
		return StoredIcon{}, err
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

func (s *IconStore) MaxBytes() int64 {
	return s.maxBytes
}

func (s *IconStore) Path(fileName string) (string, error) {
	if fileName == "" || fileName != filepath.Base(fileName) {
		return "", ErrInvalidIcon
	}
	return filepath.Join(s.dir, fileName), nil
}

func AllowedIconExtension(mimeType string) (string, bool) {
	return allowedIconExtension(mimeType)
}

func DetectIconContentType(content []byte) string {
	if isSVGContent(content) {
		return mimeTypeSVG
	}
	return http.DetectContentType(content)
}

func ValidateIconContent(content []byte) (string, string, error) {
	if len(content) == 0 {
		return "", "", ErrInvalidIcon
	}
	mimeType := DetectIconContentType(content)
	ext, ok := allowedIconExtension(mimeType)
	if !ok {
		return "", "", ErrInvalidIcon
	}
	if mimeType == mimeTypeSVG && !isSafeSVG(content) {
		return "", "", ErrInvalidIcon
	}
	return mimeType, ext, nil
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
	case mimeTypeSVG:
		return ".svg", true
	default:
		return "", false
	}
}

func isSVGContent(content []byte) bool {
	text := strings.TrimSpace(string(bytes.TrimPrefix(content, []byte{0xef, 0xbb, 0xbf})))
	if text == "" {
		return false
	}
	lower := strings.ToLower(text)
	return strings.HasPrefix(lower, "<svg") || strings.HasPrefix(lower, "<?xml") && strings.Contains(lower, "<svg")
}

func isSafeSVG(content []byte) bool {
	decoder := xml.NewDecoder(bytes.NewReader(content))
	seenRoot := false
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return seenRoot
		}
		if err != nil {
			return false
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		name := strings.ToLower(start.Name.Local)
		if !seenRoot {
			if name != "svg" {
				return false
			}
			seenRoot = true
		}
		if name == "script" {
			return false
		}
		for _, attr := range start.Attr {
			attrName := strings.ToLower(attr.Name.Local)
			attrValue := strings.ToLower(strings.TrimSpace(attr.Value))
			if strings.HasPrefix(attrName, "on") || strings.Contains(attrValue, "javascript:") {
				return false
			}
		}
	}
}
