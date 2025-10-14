package token

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Manager is responsible for retrieving and caching bearer tokens.
type Manager struct {
	client     *http.Client
	tokenURL   string
	basicAuth  string
	cachePath  string
	clock      func() time.Time
	mu         sync.Mutex
	expirySkew time.Duration
}

// NewManager constructs a new Manager.
func NewManager(client *http.Client, tokenURL, basicAuth, cachePath string) *Manager {
	return &Manager{
		client:     client,
		tokenURL:   tokenURL,
		basicAuth:  basicAuth,
		cachePath:  cachePath,
		clock:      time.Now,
		expirySkew: 30 * time.Second,
	}
}

// WithClock overrides the internal clock (used in tests).
func (m *Manager) WithClock(clock func() time.Time) {
	m.clock = clock
}

// Get returns a valid token, refreshing it when necessary.
func (m *Manager) Get(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if token, ok := m.tryLoadCached(); ok {
		if token.ExpiresAt.After(m.clock().Add(m.expirySkew)) && token.AccessToken != "" {
			return token.AccessToken, nil
		}
	}

	token, err := m.fetch(ctx)
	if err != nil {
		return "", err
	}

	if err := m.saveCached(token); err != nil {
		return "", err
	}

	return token.AccessToken, nil
}

type cachedToken struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func (m *Manager) tryLoadCached() (cachedToken, bool) {
	data, err := os.ReadFile(m.cachePath)
	if err != nil {
		return cachedToken{}, false
	}

	var token cachedToken
	if err := json.Unmarshal(data, &token); err != nil {
		return cachedToken{}, false
	}
	return token, true
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

func (m *Manager) fetch(ctx context.Context) (cachedToken, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("scope", "READSYSTEM")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return cachedToken{}, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+m.basicAuth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err := m.client.Do(req)
	if err != nil {
		return cachedToken{}, fmt.Errorf("request token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return cachedToken{}, fmt.Errorf("token endpoint returned %s: %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return cachedToken{}, fmt.Errorf("read token response: %w", err)
	}

	var parsed tokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return cachedToken{}, fmt.Errorf("parse token response: %w", err)
	}

	if parsed.AccessToken == "" || parsed.ExpiresIn <= 0 {
		return cachedToken{}, errors.New("token response missing fields")
	}

	expiry := m.clock().Add(time.Duration(parsed.ExpiresIn) * time.Second)
	return cachedToken{
		AccessToken: parsed.AccessToken,
		ExpiresAt:   expiry,
	}, nil
}

func (m *Manager) saveCached(token cachedToken) error {
	if err := os.MkdirAll(filepath.Dir(m.cachePath), 0o755); err != nil {
		return fmt.Errorf("ensure token directory: %w", err)
	}

	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("marshal token cache: %w", err)
	}

	tmp := m.cachePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write temp token cache: %w", err)
	}

	if err := os.Rename(tmp, m.cachePath); err != nil {
		return fmt.Errorf("rename token cache: %w", err)
	}

	return nil
}
