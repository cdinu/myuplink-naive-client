package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the runtime configuration for the fetcher CLI.
type Config struct {
	DeviceIDs     []string          `json:"deviceIds"`
	BasicAuth     string            `json:"basicAuth"`
	TokenFilePath string            `json:"tokenFilePath"`
	StorageRoot   string            `json:"storageRoot"`
	SQLitePath    string            `json:"sqlitePath"`
	LogsPath      string            `json:"logsPath"`
	APIBaseURL    string            `json:"apiBaseUrl"`
	ParameterSets map[string]string `json:"parameterSets"`
}

// Load reads and unmarshals the configuration from the provided path.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg.normalize(), nil
}

func (c Config) validate() error {
	if len(c.DeviceIDs) == 0 {
		return fmt.Errorf("config: deviceIds is required")
	}
	switch "" {
	case c.BasicAuth:
		return fmt.Errorf("config: basicAuth is required")
	case c.TokenFilePath:
		return fmt.Errorf("config: tokenFilePath is required")
	case c.StorageRoot:
		return fmt.Errorf("config: storageRoot is required")
	case c.SQLitePath:
		return fmt.Errorf("config: sqlitePath is required")
	case c.LogsPath:
		return fmt.Errorf("config: logsPath is required")
	}
	return nil
}

func (c Config) normalize() Config {
	if c.APIBaseURL == "" {
		c.APIBaseURL = "https://api.myuplink.com"
	}

	c.StorageRoot = filepath.Clean(c.StorageRoot)
	c.TokenFilePath = filepath.Clean(c.TokenFilePath)
	c.SQLitePath = filepath.Clean(c.SQLitePath)
	c.LogsPath = filepath.Clean(c.LogsPath)

	return c
}
