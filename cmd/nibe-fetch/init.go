package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cdinu/myuplink-naive-client/internal/config"
)

func initializeWorkspace(configPath string) (string, error) {
	if configPath == "" {
		configPath = "config.json"
	}

	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		return "", fmt.Errorf("resolve config path: %w", err)
	}

	configDir := filepath.Dir(absConfig)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", fmt.Errorf("ensure config directory: %w", err)
	}

	defaults := defaultConfig(configDir)

	configExists := true
	if _, err := os.Stat(absConfig); errors.Is(err, os.ErrNotExist) {
		configExists = false
		data, marshalErr := json.MarshalIndent(defaults, "", "  ")
		if marshalErr != nil {
			return "", fmt.Errorf("marshal default config: %w", marshalErr)
		}
		if writeErr := os.WriteFile(absConfig, append(data, '\n'), 0o644); writeErr != nil {
			return "", fmt.Errorf("write config: %w", writeErr)
		}
	} else if err != nil {
		return "", fmt.Errorf("stat config: %w", err)
	}

	dirs := []string{
		filepath.Dir(defaults.TokenFilePath),
		defaults.StorageRoot,
		filepath.Dir(defaults.SQLitePath),
		defaults.LogsPath,
	}
	uniqueDirs := uniqueStrings(dirs)

	for _, dir := range uniqueDirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("ensure directory %s: %w", dir, err)
		}
	}

	var sb strings.Builder
	if configExists {
		sb.WriteString(fmt.Sprintf("Config already exists at %s\n", absConfig))
	} else {
		sb.WriteString(fmt.Sprintf("Created config at %s\n", absConfig))
	}
	sb.WriteString("Ensured directories:\n")
	for _, dir := range uniqueDirs {
		sb.WriteString(fmt.Sprintf("  - %s\n", dir))
	}

	return sb.String(), nil
}

func defaultConfig(base string) config.Config {
	cacheDir := filepath.Join(base, "cache")
	dataDir := filepath.Join(base, "data")
	stateDir := filepath.Join(base, "state")
	logsDir := filepath.Join(base, "logs")

	return config.Config{
		DeviceID:      "REPLACE_WITH_DEVICE_ID",
		BasicAuth:     "base64(client_id:client_secret)",
		TokenFilePath: filepath.Join(cacheDir, "token.json"),
		StorageRoot:   dataDir,
		SQLitePath:    filepath.Join(stateDir, "nibe.sqlite"),
		LogsPath:      logsDir,
		APIBaseURL:    "https://api.myuplink.com",
	}
}

func uniqueStrings(items []string) []string {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		set[filepath.Clean(item)] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}
