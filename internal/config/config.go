package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App    AppConfig    `mapstructure:"app"`
	HTTP   HTTPConfig   `mapstructure:"http"`
	Log    LogConfig    `mapstructure:"log"`
	Upload UploadConfig `mapstructure:"upload"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
}

type HTTPConfig struct {
	Addr              string `mapstructure:"addr"`
	ReadHeaderTimeout string `mapstructure:"read_header_timeout"`
	ShutdownTimeout   string `mapstructure:"shutdown_timeout"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

type UploadConfig struct {
	Dir      string `mapstructure:"dir"`
	MaxBytes int64  `mapstructure:"max_bytes"`
}

func Load() (Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")
	v.SetEnvPrefix("NAVBOX")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "navbox")
	v.SetDefault("http.addr", ":8080")
	v.SetDefault("http.read_header_timeout", "5s")
	v.SetDefault("http.shutdown_timeout", "10s")
	v.SetDefault("log.level", "info")
	v.SetDefault("upload.dir", "./data/uploads")
	v.SetDefault("upload.max_bytes", 1048576)
}

func (c Config) Validate() error {
	if c.App.Name == "" {
		return errors.New("app name is required")
	}
	if c.HTTP.Addr == "" {
		return errors.New("http addr is required")
	}
	if _, err := time.ParseDuration(c.HTTP.ReadHeaderTimeout); err != nil {
		return fmt.Errorf("invalid http read_header_timeout: %w", err)
	}
	if _, err := time.ParseDuration(c.HTTP.ShutdownTimeout); err != nil {
		return fmt.Errorf("invalid http shutdown_timeout: %w", err)
	}
	if c.Upload.Dir == "" {
		return errors.New("upload dir is required")
	}
	if c.Upload.MaxBytes <= 0 {
		return errors.New("upload max_bytes must be greater than zero")
	}
	return nil
}
