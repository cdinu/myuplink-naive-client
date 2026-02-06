package app_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cdinu/myuplink-naive-client/internal/app"
	"github.com/cdinu/myuplink-naive-client/internal/config"
	_ "modernc.org/sqlite"
)

func TestRun_PersistsTelemetry(t *testing.T) {
	t.Helper()

	deviceID := "device-test-123"
	telemetryPayload := `[{"category":"Group","parameterId":"40004","parameterName":"Outdoor temperature","parameterUnit":"°C","writable":false,"timestamp":"2025-10-14T04:21:37+00:00","value":14.8,"strVal":"14.8°C","smartHomeCategories":["sh-outdoorTemp"],"minValue":-40,"maxValue":70,"stepValue":0.1,"enumValues":[],"scaleValue":"0.1","zoneId":null},{"category":"Group","parameterId":"40008","parameterName":"Supply line (BT2)","parameterUnit":"°C","writable":false,"timestamp":"2025-10-14T09:04:26+00:00","value":27.4,"strVal":"27.4°C","smartHomeCategories":[],"minValue":null,"maxValue":null,"stepValue":0.1,"enumValues":[],"scaleValue":"0.1","zoneId":null}]`

	tokenResponse := map[string]any{
		"access_token": "test-token",
		"expires_in":   3600,
		"token_type":   "Bearer",
		"scope":        "READSYSTEM",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenResponse)
	})
	mux.HandleFunc("/v3/devices/"+deviceID+"/points", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("missing bearer auth, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, telemetryPayload)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	root := t.TempDir()

	cfg := config.Config{
		DeviceIDs:     []string{deviceID},
		BasicAuth:     "dummy",
		TokenFilePath: filepath.Join(root, "cache", "token.json"),
		StorageRoot:   filepath.Join(root, "data"),
		SQLitePath:    filepath.Join(root, "state", "nibe.sqlite"),
		LogsPath:      filepath.Join(root, "logs"),
		APIBaseURL:    server.URL,
	}

	logger := log.New(io.Discard, "", log.LstdFlags)

	ctx := context.Background()
	if err := app.Run(ctx, cfg, "", logger); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if _, err := os.Stat(cfg.TokenFilePath); err != nil {
		t.Fatalf("token file not created: %v", err)
	}

	day := time.Now().UTC().Format("2006-01-02")
	jsonlPath := filepath.Join(cfg.StorageRoot, fmt.Sprintf("%s.%s.jsonl", deviceID, day))
	content, err := os.ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("read jsonl: %v", err)
	}
	line := strings.TrimSuffix(string(content), "\n")
	if line != telemetryPayload {
		t.Fatalf("jsonl payload mismatch\nexpected: %s\nactual:   %s", telemetryPayload, line)
	}

	db, err := sql.Open("sqlite", cfg.SQLitePath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT device_id, parameter_id, value, raw_json FROM telemetry ORDER BY id`)
	if err != nil {
		t.Fatalf("query telemetry: %v", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var device, parameterID, value, raw string
		if err := rows.Scan(&device, &parameterID, &value, &raw); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if device != deviceID {
			t.Fatalf("unexpected device_id %q", device)
		}
		if !strings.Contains(telemetryPayload, raw) {
			t.Fatalf("raw json not found in payload: %s", raw)
		}
		count++
		if parameterID == "40004" && value != "14.8" {
			t.Fatalf("unexpected value for 40004: %s", value)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("row iteration: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 rows, got %d", count)
	}
}
