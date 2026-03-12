package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func setupTest(t *testing.T) string {
	t.Helper()
	viper.Reset()
	tmpDir := t.TempDir()
	configDirOverride = tmpDir
	t.Cleanup(func() {
		configDirOverride = ""
		viper.Reset()
	})
	return tmpDir
}

func TestLoadConfig(t *testing.T) {
	tmpDir := setupTest(t)

	// Write a config file
	configContent := "api-key: test-key-123\napi-url: https://custom.api.com\n"
	err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0o600)
	require.NoError(t, err)

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "test-key-123", cfg.APIKey)
	require.Equal(t, "https://custom.api.com", cfg.APIURL)
}

func TestLoadConfigMissing(t *testing.T) {
	setupTest(t)

	cfg, err := Load()
	require.NoError(t, err)
	require.Empty(t, cfg.APIKey)
	require.Equal(t, "https://api.revenium.ai/profitstream", cfg.APIURL)
}

func TestLoadConfigDefault(t *testing.T) {
	tmpDir := setupTest(t)

	// Config with only api-key, no api-url
	configContent := "api-key: some-key\n"
	err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0o600)
	require.NoError(t, err)

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "https://api.revenium.ai/profitstream", cfg.APIURL)
}

func TestSetConfig(t *testing.T) {
	tmpDir := setupTest(t)

	err := Set("api-key", "my-secret-key")
	require.NoError(t, err)

	// Verify file was written
	data, err := os.ReadFile(filepath.Join(tmpDir, "config.yaml"))
	require.NoError(t, err)
	require.Contains(t, string(data), "my-secret-key")

	// Set api-url too
	viper.Reset()
	err = Set("api-url", "https://other.api.com")
	require.NoError(t, err)

	data, err = os.ReadFile(filepath.Join(tmpDir, "config.yaml"))
	require.NoError(t, err)
	require.Contains(t, string(data), "https://other.api.com")
}

func TestSetConfigCreatesDir(t *testing.T) {
	viper.Reset()
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested", "config")
	configDirOverride = nestedDir
	t.Cleanup(func() {
		configDirOverride = ""
		viper.Reset()
	})

	err := Set("api-key", "new-key")
	require.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(nestedDir)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestEnvOverrideAPIKey(t *testing.T) {
	tmpDir := setupTest(t)

	// Write config file with one value
	configContent := "api-key: file-key\n"
	err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0o600)
	require.NoError(t, err)

	// Set env var to override
	t.Setenv("REVENIUM_API_KEY", "env-key-override")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "env-key-override", cfg.APIKey)
}

func TestEnvOverrideAPIURL(t *testing.T) {
	tmpDir := setupTest(t)

	// Write config file with one value
	configContent := "api-url: https://file.api.com\n"
	err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0o600)
	require.NoError(t, err)

	// Set env var to override
	t.Setenv("REVENIUM_API_URL", "https://env.api.com")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "https://env.api.com", cfg.APIURL)
}
