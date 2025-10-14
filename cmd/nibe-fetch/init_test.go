package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cdinu/myuplink-naive-client/internal/config"
)

func TestInitializeWorkspaceCreatesConfigAndDirs(t *testing.T) {
	t.Helper()

	tempRoot := t.TempDir()
	configPath := filepath.Join(tempRoot, "env", "config.json")

	summary, err := initializeWorkspace(configPath)
	if err != nil {
		t.Fatalf("initializeWorkspace error: %v", err)
	}
	if !strings.Contains(summary, "Created config") {
		t.Fatalf("summary missing creation message: %q", summary)
	}

	data, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse written config: %v", err)
	}

	baseDir := filepath.Dir(configPath)
	expectedSet := map[string]struct{}{
		filepath.Join(baseDir, "cache"): {},
		filepath.Join(baseDir, "data"):  {},
		filepath.Join(baseDir, "logs"):  {},
		filepath.Join(baseDir, "state"): {},
	}
	dirs := []string{
		filepath.Dir(cfg.TokenFilePath),
		cfg.StorageRoot,
		filepath.Dir(cfg.SQLitePath),
		cfg.LogsPath,
	}

	for _, dir := range dirs {
		dir = filepath.Clean(dir)
		if _, ok := expectedSet[dir]; !ok {
			t.Fatalf("unexpected directory %s", dir)
		}
		if _, err := os.Stat(dir); err != nil {
			t.Fatalf("expected directory %s: %v", dir, err)
		}
	}

	// Second run should not overwrite existing config.
	summary, err = initializeWorkspace(configPath)
	if err != nil {
		t.Fatalf("second initializeWorkspace error: %v", err)
	}
	if !strings.Contains(summary, "Config already exists") {
		t.Fatalf("summary missing existing message: %q", summary)
	}
}
