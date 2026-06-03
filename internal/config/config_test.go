package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
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

// TestSetDoesNotPersistOverrideFlags is the CR-01 regression test. It mirrors the
// real cmd/root.go init() wiring by binding the five override pflags to the
// GLOBAL Viper instance and setting --api-key to a secret. It then calls
// Set("team-id", "X") and asserts the written file contains neither the one-shot
// secret nor any (empty) override key — only the keys legitimately set. Before
// the fix, Set() wrote through the global Viper whose WriteConfigAs serialized
// AllSettings(), persisting the secret and injecting empty override keys.
func TestSetDoesNotPersistOverrideFlags(t *testing.T) {
	tmpDir := setupTest(t) // resets global viper before and after via t.Cleanup

	// Reproduce init()'s global-Viper binding of the five override flags.
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	for _, name := range []string{"api-key", "api-url", "team-id", "tenant-id", "owner-id"} {
		fs.String(name, "", "")
	}
	// Simulate a one-shot --api-key SECRET passed on the command line.
	require.NoError(t, fs.Set("api-key", "SUPER-SECRET"))
	for _, name := range []string{"api-key", "api-url", "team-id", "tenant-id", "owner-id"} {
		require.NoError(t, viper.BindPFlag(name, fs.Lookup(name)))
	}

	// Sanity-check the binding is live on the global instance.
	require.Equal(t, "SUPER-SECRET", viper.GetString("api-key"))

	err := Set("team-id", "X")
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(tmpDir, "config.yaml"))
	require.NoError(t, err)
	contents := string(data)

	// The legitimately-set key must be present.
	require.Contains(t, contents, "team-id")
	require.Contains(t, contents, "X")

	// The one-shot secret must NOT be persisted.
	require.NotContains(t, contents, "SUPER-SECRET")
	require.NotContains(t, contents, "api-key")

	// No empty override keys should be injected from unchanged pflag defaults.
	require.NotContains(t, contents, "api-url")
	require.NotContains(t, contents, "tenant-id")
	require.NotContains(t, contents, "owner-id")
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

// TestFlagOverridesEnvAndFile proves Viper's flag > env > file precedence
// (CFGO-06) at the resolution layer: a bound, changed pflag wins over both the
// REVENIUM_API_KEY env var and the config file value. This mirrors the
// viper.BindPFlag wiring in cmd/root.go init(); removing that binding (or
// failing to mark the flag changed) makes this test fail.
func TestFlagOverridesEnvAndFile(t *testing.T) {
	tmpDir := setupTest(t)

	configContent := "api-key: file-key\n"
	err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0o600)
	require.NoError(t, err)

	t.Setenv("REVENIUM_API_KEY", "env-key")

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("api-key", "", "")
	require.NoError(t, fs.Set("api-key", "flag-key"))
	require.NoError(t, viper.BindPFlag("api-key", fs.Lookup("api-key")))

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "flag-key", cfg.APIKey, "flag should win over env and file")
}

// TestUnsetFlagFallsThroughToEnv proves that a bound pflag that was never set
// (Changed == false) does NOT override env/file resolution (CFGO-06 second
// half, D-07: no special-casing). The env value must still win over the file.
func TestUnsetFlagFallsThroughToEnv(t *testing.T) {
	tmpDir := setupTest(t)

	configContent := "api-key: file-key\n"
	err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0o600)
	require.NoError(t, err)

	t.Setenv("REVENIUM_API_KEY", "env-key")

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("api-key", "", "")
	// Intentionally do NOT call fs.Set — the flag is unchanged.
	require.NoError(t, viper.BindPFlag("api-key", fs.Lookup("api-key")))

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "env-key", cfg.APIKey, "an unset flag must fall through to the env value")
}

// TestTeamIDFlagOverridesEnvAndFile generalizes the BindPFlag precedence proof
// beyond api-key. It uses --team-id (Viper key "team-id", env REVENIUM_TEAM_ID,
// cfg.TeamID) to confirm the binding mechanism works uniformly across all five
// override flags (CFGO-02..05 resolution for non-api-key flags).
func TestTeamIDFlagOverridesEnvAndFile(t *testing.T) {
	tmpDir := setupTest(t)

	// File value: team-id = file-team
	configContent := "team-id: file-team\n"
	err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0o600)
	require.NoError(t, err)

	// Env value: REVENIUM_TEAM_ID = env-team
	t.Setenv("REVENIUM_TEAM_ID", "env-team")

	// Flag value: --team-id flag-team (marked changed via fs.Set)
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("team-id", "", "")
	require.NoError(t, fs.Set("team-id", "flag-team"))
	require.NoError(t, viper.BindPFlag("team-id", fs.Lookup("team-id")))

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "flag-team", cfg.TeamID, "flag should win over env and file for team-id")
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
