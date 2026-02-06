package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// ErrParsePoints indicates the response could not be parsed as telemetry points.
var ErrParsePoints = errors.New("parse telemetry response")

// Client retrieves telemetry data for devices.
type Client struct {
	client  *http.Client
	baseURL string
}

// NewClient constructs a new telemetry client.
func NewClient(client *http.Client, baseURL string) *Client {
	base := strings.TrimSuffix(baseURL, "/")
	return &Client{
		client:  client,
		baseURL: base,
	}
}

// FetchPoints fetches the telemetry JSON payload and splits it into per-point raw messages.
func (c *Client) FetchPoints(ctx context.Context, deviceID, token string, parameters string) ([]byte, []json.RawMessage, error) {
	pointsURL := c.baseURL + path.Join("/v3/devices/", deviceID, "/points")
	if parameters != "" {
		u, err := url.Parse(pointsURL)
		if err != nil {
			return nil, nil, fmt.Errorf("parse URL: %w", err)
		}
		q := u.Query()
		q.Set("parameters", parameters)
		u.RawQuery = q.Encode()
		pointsURL = u.String()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pointsURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create telemetry request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("Accept-Language", "en-US")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request telemetry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, nil, fmt.Errorf("telemetry endpoint returned %s: %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read telemetry response: %w", err)
	}

	var rawPoints []json.RawMessage
	if err := json.Unmarshal(body, &rawPoints); err != nil {
		return body, nil, fmt.Errorf("%w: %v", ErrParsePoints, err)
	}

	return body, rawPoints, nil
}
