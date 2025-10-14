package token_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cdinu/myuplink-naive-client/internal/token"
)

func TestManager_ReusesCachedToken(t *testing.T) {
	t.Helper()

	var calls int32
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&calls) > 0 {
			t.Fatalf("token endpoint called more than once")
		}
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"access_token": "cached-token",
			"expires_in":   3600,
			"token_type":   "Bearer",
			"scope":        "READSYSTEM",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	cachePath := filepath.Join(t.TempDir(), "token.json")
	client := server.Client()
	client.Timeout = 5 * time.Second

	manager := token.NewManager(client, server.URL+"/oauth/token", "ignored", cachePath)

	now := time.Unix(0, 0)
	manager.WithClock(func() time.Time { return now })

	ctx := context.Background()
	tokenValue, err := manager.Get(ctx)
	if err != nil {
		t.Fatalf("first Get: %v", err)
	}
	if tokenValue != "cached-token" {
		t.Fatalf("unexpected token value %q", tokenValue)
	}

	now = now.Add(5 * time.Second)
	tokenValue, err = manager.Get(ctx)
	if err != nil {
		t.Fatalf("second Get: %v", err)
	}
	if tokenValue != "cached-token" {
		t.Fatalf("unexpected token on second get %q", tokenValue)
	}
}
