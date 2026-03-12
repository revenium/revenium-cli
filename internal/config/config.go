// Package config manages CLI configuration stored at ~/.config/revenium/config.yaml.
// It supports loading from file, setting values, and environment variable overrides.
package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// configDirOverride allows tests to redirect config operations to a temp directory.
var configDirOverride string

// Config holds the CLI configuration values.
type Config struct {
	APIKey string
	APIURL string
}

// configDir returns the configuration directory path.
// Uses ~/.config/revenium (XDG standard, not os.UserConfigDir which returns
// ~/Library/Application Support on macOS).
func configDir() (string, error) {
	if configDirOverride != "" {
		return configDirOverride, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "revenium"), nil
}

// configPath returns the full path to the config file.
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load reads configuration from the config file and environment variables.
// Environment variables REVENIUM_API_KEY and REVENIUM_API_URL override file values.
// Returns a Config with defaults applied if no file exists.
func Load() (*Config, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(dir)

	viper.SetEnvPrefix("REVENIUM")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	_ = viper.BindEnv("api-key")
	_ = viper.BindEnv("api-url")

	viper.SetDefault("api-url", "https://api.revenium.ai/profitstream")

	if err := viper.ReadInConfig(); err != nil {
		var configNotFound viper.ConfigFileNotFoundError
		if !errors.As(err, &configNotFound) {
			// Only return error if it's not a missing file
			if !os.IsNotExist(err) {
				return nil, err
			}
		}
	}

	return &Config{
		APIKey: viper.GetString("api-key"),
		APIURL: viper.GetString("api-url"),
	}, nil
}

// Set writes a key-value pair to the config file, creating the directory if needed.
func Set(key, value string) error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	path, err := configPath()
	if err != nil {
		return err
	}

	viper.SetConfigFile(path)

	// Read existing config, ignore if file doesn't exist
	_ = viper.ReadInConfig()

	viper.Set(key, value)

	return viper.WriteConfigAs(path)
}
