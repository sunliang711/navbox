package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	HTTP     HTTPConfig     `mapstructure:"http"`
	Log      LogConfig      `mapstructure:"log"`
	Database DatabaseConfig `mapstructure:"database"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Upload   UploadConfig   `mapstructure:"upload"`
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

type DatabaseConfig struct {
	DSN             string `mapstructure:"dsn"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime string `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime string `mapstructure:"conn_max_idle_time"`
	ConnectTimeout  string `mapstructure:"connect_timeout"`
}

type AuthConfig struct {
	SessionTTL            string `mapstructure:"session_ttl"`
	CookieName            string `mapstructure:"cookie_name"`
	CookieSecure          bool   `mapstructure:"cookie_secure"`
	InitialPasswordLength int    `mapstructure:"initial_password_length"`
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
	v.SetDefault("database.dsn", "")
	v.SetDefault("database.max_open_conns", 20)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "1h")
	v.SetDefault("database.conn_max_idle_time", "30m")
	v.SetDefault("database.connect_timeout", "5s")
	v.SetDefault("auth.session_ttl", "24h")
	v.SetDefault("auth.cookie_name", "navbox_admin_session")
	v.SetDefault("auth.cookie_secure", false)
	v.SetDefault("auth.initial_password_length", 16)
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
	if c.Database.DSN == "" {
		return errors.New("database dsn is required")
	}
	if c.Database.MaxOpenConns <= 0 {
		return errors.New("database max_open_conns must be greater than zero")
	}
	if c.Database.MaxIdleConns <= 0 {
		return errors.New("database max_idle_conns must be greater than zero")
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return errors.New("database max_idle_conns must be less than or equal to max_open_conns")
	}
	if _, err := time.ParseDuration(c.Database.ConnMaxLifetime); err != nil {
		return fmt.Errorf("invalid database conn_max_lifetime: %w", err)
	}
	if _, err := time.ParseDuration(c.Database.ConnMaxIdleTime); err != nil {
		return fmt.Errorf("invalid database conn_max_idle_time: %w", err)
	}
	if _, err := time.ParseDuration(c.Database.ConnectTimeout); err != nil {
		return fmt.Errorf("invalid database connect_timeout: %w", err)
	}
	if _, err := time.ParseDuration(c.Auth.SessionTTL); err != nil {
		return fmt.Errorf("invalid auth session_ttl: %w", err)
	}
	if c.Auth.CookieName == "" {
		return errors.New("auth cookie_name is required")
	}
	if c.Auth.InitialPasswordLength < 16 {
		return errors.New("auth initial_password_length must be at least 16")
	}
	if c.Upload.Dir == "" {
		return errors.New("upload dir is required")
	}
	if c.Upload.MaxBytes <= 0 {
		return errors.New("upload max_bytes must be greater than zero")
	}
	return nil
}
