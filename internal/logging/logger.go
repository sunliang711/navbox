package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"

	"navbox/internal/config"
)

func NewLogger(cfg config.Config) (zerolog.Logger, error) {
	level, err := zerolog.ParseLevel(cfg.Log.Level)
	if err != nil {
		return zerolog.Logger{}, fmt.Errorf("parse log level: %w", err)
	}
	zerolog.SetGlobalLevel(level)
	zerolog.CallerMarshalFunc = trimCallerPath

	return zerolog.New(os.Stdout).
		With().
		Timestamp().
		Caller().
		Str("app", cfg.App.Name).
		Logger(), nil
}

func trimCallerPath(_ uintptr, file string, line int) string {
	clean := filepath.ToSlash(file)
	parts := strings.Split(clean, "/")
	if len(parts) > 2 {
		clean = strings.Join(parts[len(parts)-2:], "/")
	}
	return fmt.Sprintf("%s:%d", clean, line)
}
