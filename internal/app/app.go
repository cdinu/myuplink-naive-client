package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cdinu/myuplink-naive-client/internal/config"
	"github.com/cdinu/myuplink-naive-client/internal/storage"
	"github.com/cdinu/myuplink-naive-client/internal/telemetry"
	"github.com/cdinu/myuplink-naive-client/internal/token"
)

// Run executes a single telemetry collection cycle.
func Run(ctx context.Context, cfg config.Config, logger *log.Logger) error {
	if err := ensurePaths(cfg); err != nil {
		return err
	}

	httpClient := &http.Client{
		Timeout: 20 * time.Second,
	}

	tokenURL := strings.TrimSuffix(cfg.APIBaseURL, "/") + "/oauth/token"
	tokenManager := token.NewManager(httpClient, tokenURL, cfg.BasicAuth, cfg.TokenFilePath)

	db, err := storage.OpenSQLite(cfg.SQLitePath)
	if err != nil {
		return err
	}
	defer db.Close()

	tkn, err := tokenManager.Get(ctx)
	if err != nil {
		return fmt.Errorf("acquire token: %w", err)
	}
	logger.Printf("token acquired; expires cached at %s", cfg.TokenFilePath)

	telemetryClient := telemetry.NewClient(httpClient, cfg.APIBaseURL)
	body, rawPoints, err := telemetryClient.FetchPoints(ctx, cfg.DeviceID, tkn)
	if err != nil {
		if !errors.Is(err, telemetry.ErrParsePoints) {
			return fmt.Errorf("fetch telemetry: %w", err)
		}
		logger.Printf("telemetry parse error: %v", err)
	}

	day := time.Now().UTC().Format("2006-01-02")
	jsonlPath, err := storage.AppendJSONL(cfg.StorageRoot, cfg.DeviceID, day, body)
	if err != nil {
		return err
	}
	logger.Printf("appended telemetry payload to %s", jsonlPath)

	if len(rawPoints) == 0 {
		logger.Printf("no telemetry points parsed; skipping sqlite insert")
		return nil
	}

	records, err := convertRecords(cfg.DeviceID, rawPoints)
	if err != nil {
		logger.Printf("telemetry conversion error: %v", err)
		return nil
	}

	if err := db.InsertTelemetry(ctx, records); err != nil {
		return err
	}
	logger.Printf("inserted %d telemetry rows", len(records))

	return nil
}

func ensurePaths(cfg config.Config) error {
	if err := os.MkdirAll(cfg.LogsPath, 0o755); err != nil {
		return fmt.Errorf("ensure logs path: %w", err)
	}
	if err := os.MkdirAll(cfg.StorageRoot, 0o755); err != nil {
		return fmt.Errorf("ensure storage root: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.TokenFilePath), 0o755); err != nil {
		return fmt.Errorf("ensure token dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o755); err != nil {
		return fmt.Errorf("ensure sqlite dir: %w", err)
	}
	return nil
}

type telemetryPoint struct {
	Category      string          `json:"category"`
	ParameterID   string          `json:"parameterId"`
	ParameterName string          `json:"parameterName"`
	ParameterUnit string          `json:"parameterUnit"`
	Writable      *bool           `json:"writable"`
	Timestamp     string          `json:"timestamp"`
	Value         json.RawMessage `json:"value"`
	StrVal        string          `json:"strVal"`
	SmartHomeCats []string        `json:"smartHomeCategories"`
	MinValue      *float64        `json:"minValue"`
	MaxValue      *float64        `json:"maxValue"`
	StepValue     *float64        `json:"stepValue"`
	EnumValues    json.RawMessage `json:"enumValues"`
	ScaleValue    string          `json:"scaleValue"`
	ZoneID        string          `json:"zoneId"`
}

func convertRecords(deviceID string, rawPoints []json.RawMessage) ([]storage.TelemetryRecord, error) {
	storedAt := time.Now()
	records := make([]storage.TelemetryRecord, 0, len(rawPoints))

	for _, raw := range rawPoints {
		var point telemetryPoint
		if err := json.Unmarshal(raw, &point); err != nil {
			// Skip invalid points but continue processing the rest.
			continue
		}

		smartCats := "[]"
		if point.SmartHomeCats != nil {
			buf, err := json.Marshal(point.SmartHomeCats)
			if err != nil {
				return nil, fmt.Errorf("marshal smart home categories: %w", err)
			}
			smartCats = string(buf)
		}

		enumVals := "[]"
		if len(point.EnumValues) > 0 {
			enumVals = string(point.EnumValues)
		}

		value := "null"
		if len(point.Value) > 0 {
			value = string(point.Value)
		}

		rec := storage.TelemetryRecord{
			DeviceID:            deviceID,
			StoredAt:            storedAt,
			Category:            point.Category,
			ParameterID:         point.ParameterID,
			ParameterName:       point.ParameterName,
			ParameterUnit:       point.ParameterUnit,
			Writable:            point.Writable,
			ReadingTimestamp:    point.Timestamp,
			Value:               value,
			StrVal:              point.StrVal,
			SmartHomeCategories: smartCats,
			MinValue:            point.MinValue,
			MaxValue:            point.MaxValue,
			StepValue:           point.StepValue,
			EnumValues:          enumVals,
			ScaleValue:          point.ScaleValue,
			ZoneID:              point.ZoneID,
			RawJSON:             string(raw),
		}

		records = append(records, rec)
	}

	return records, nil
}
