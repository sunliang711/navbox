package service

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"navbox/internal/dto"
	"navbox/internal/storage"
)

func TestLocalIconFileName(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
		ok    bool
	}{
		{name: "uploads path", value: "/uploads/icon.png", want: "icon.png", ok: true},
		{name: "relative uploads path", value: "uploads/icon.png", want: "icon.png", ok: true},
		{name: "plain file", value: "icon.webp", want: "icon.webp", ok: true},
		{name: "online url", value: "https://example.com/icon.png", ok: false},
		{name: "nested path", value: "/uploads/a/icon.png", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := localIconFileName(tt.value)

			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseArchiveSitesRejectsDuplicateID(t *testing.T) {
	id := "00000000-0000-0000-0000-000000000001"

	_, _, _, err := parseArchiveSites([]dto.ArchiveSite{
		{ID: id, Title: "A", DefaultURL: "https://a.example.com"},
		{ID: id, Title: "B", DefaultURL: "https://b.example.com"},
	})

	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestReadArchiveRejectsTraversal(t *testing.T) {
	svc := newTestImportExportService(t)
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	fileWriter, err := writer.Create("../navbox.json")
	require.NoError(t, err)
	_, err = fileWriter.Write([]byte(`{}`))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	_, _, err = svc.readArchive(buf.Bytes())

	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestValidateArchiveIcon(t *testing.T) {
	svc := newTestImportExportService(t)
	content := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x04, 0x00, 0x00, 0x00, 0xb5, 0x1c, 0x0c,
		0x02, 0x00, 0x00, 0x00, 0x0b, 0x49, 0x44, 0x41,
		0x54, 0x78, 0xda, 0x63, 0xfc, 0xff, 0x1f, 0x00,
		0x03, 0x03, 0x02, 0x00, 0xef, 0xbf, 0xa7, 0xdb,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44,
		0xae, 0x42, 0x60, 0x82,
	}
	sum := sha256.Sum256(content)
	shaValue := hex.EncodeToString(sum[:])

	err := svc.validateArchiveIcon(dto.ArchiveIcon{
		FileName:  shaValue + ".png",
		SHA256:    shaValue,
		MIMEType:  "image/png",
		SizeBytes: int64(len(content)),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, content)

	require.NoError(t, err)
}

func newTestImportExportService(t *testing.T) *importExportService {
	t.Helper()

	store, err := storage.NewIconStore(t.TempDir(), 1024*1024)
	require.NoError(t, err)

	return &importExportService{store: store}
}
