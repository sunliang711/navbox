package storage

import (
	"bytes"
	"errors"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIconStoreSavePNG(t *testing.T) {
	store, err := NewIconStore(t.TempDir(), 1024)
	require.NoError(t, err)

	icon, err := store.Save(newFakeMultipartFile([]byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d,
	}))

	require.NoError(t, err)
	require.Equal(t, "image/png", icon.MIMEType)
	require.Contains(t, icon.FileName, ".png")
	require.NotEmpty(t, icon.SHA256)
}

func TestIconStoreRejectsText(t *testing.T) {
	store, err := NewIconStore(t.TempDir(), 1024)
	require.NoError(t, err)

	_, err = store.Save(newFakeMultipartFile([]byte("not an image")))

	require.ErrorIs(t, err, ErrInvalidIcon)
}

func TestIconStoreRejectsLargeFile(t *testing.T) {
	store, err := NewIconStore(t.TempDir(), 4)
	require.NoError(t, err)

	_, err = store.Save(newFakeMultipartFile([]byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d,
	}))

	require.True(t, errors.Is(err, ErrIconTooLarge) || errors.Is(err, ErrInvalidIcon))
}

type fakeMultipartFile struct {
	reader *bytes.Reader
}

func newFakeMultipartFile(data []byte) *fakeMultipartFile {
	return &fakeMultipartFile{reader: bytes.NewReader(data)}
}

func (f *fakeMultipartFile) Read(p []byte) (int, error) {
	return f.reader.Read(p)
}

func (f *fakeMultipartFile) ReadAt(p []byte, off int64) (int, error) {
	return f.reader.ReadAt(p, off)
}

func (f *fakeMultipartFile) Seek(offset int64, whence int) (int64, error) {
	return f.reader.Seek(offset, whence)
}

func (f *fakeMultipartFile) Close() error {
	return nil
}

var _ multipart.File = (*fakeMultipartFile)(nil)
