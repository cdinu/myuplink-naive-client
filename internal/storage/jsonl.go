package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// AppendJSONL appends a raw JSON payload as a single line to a jsonl file.
func AppendJSONL(dir, deviceID string, day string, payload []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("ensure storage directory: %w", err)
	}

	filename := fmt.Sprintf("%s.%s.jsonl", deviceID, day)
	path := filepath.Join(dir, filename)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("open storage file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(payload, '\n')); err != nil {
		return "", fmt.Errorf("append storage file: %w", err)
	}

	return path, nil
}
