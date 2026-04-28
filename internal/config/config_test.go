package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidateRestoreMode(t *testing.T) {
	cfg := testConfig()
	cfg.Restore = RestoreConfig{
		Mode:  RestoreModeAdminPassword,
		Token: "0123456789abcdef0123456789abcdef",
	}

	require.NoError(t, cfg.Validate())
}

func TestConfigValidateRestoreModeRequiresStrongToken(t *testing.T) {
	cfg := testConfig()
	cfg.Restore = RestoreConfig{
		Mode:  RestoreModeAdminPassword,
		Token: "short",
	}

	err := cfg.Validate()

	require.ErrorContains(t, err, "restore token")
}

func TestConfigValidateRejectsInvalidRestoreMode(t *testing.T) {
	cfg := testConfig()
	cfg.Restore = RestoreConfig{
		Mode:  "invalid",
		Token: "0123456789abcdef0123456789abcdef",
	}

	err := cfg.Validate()

	require.ErrorContains(t, err, "restore mode")
}

func testConfig() Config {
	return Config{
		App: AppConfig{
			Name: "navbox",
		},
		HTTP: HTTPConfig{
			Addr:              ":8037",
			ReadHeaderTimeout: "5s",
			ShutdownTimeout:   "10s",
		},
		Database: DatabaseConfig{
			DSN:             "host=localhost user=navbox password=navbox dbname=navbox port=5432 sslmode=disable",
			MaxOpenConns:    20,
			MaxIdleConns:    5,
			ConnMaxLifetime: "1h",
			ConnMaxIdleTime: "30m",
			ConnectTimeout:  "5s",
		},
		Auth: AuthConfig{
			SessionTTL:            "24h",
			CookieName:            "navbox_admin_session",
			InitialPasswordLength: 16,
		},
		Upload: UploadConfig{
			Dir:      "./data/uploads",
			MaxBytes: 1048576,
		},
	}
}
