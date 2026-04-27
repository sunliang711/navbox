package storage

import "navbox/internal/config"

func NewIconStoreFromConfig(cfg config.Config) (*IconStore, error) {
	return NewIconStore(cfg.Upload.Dir, cfg.Upload.MaxBytes)
}
