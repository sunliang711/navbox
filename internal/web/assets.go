package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var embeddedFiles embed.FS

type Assets struct {
	FS http.FileSystem
}

func NewAssets() (Assets, error) {
	dist, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		return Assets{}, err
	}
	return Assets{FS: http.FS(dist)}, nil
}
