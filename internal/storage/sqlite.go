package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps an sqlite database connection.
type DB struct {
	sql *sql.DB
}

// TelemetryRecord represents a single telemetry point to be persisted.
type TelemetryRecord struct {
	DeviceID            string
	StoredAt            time.Time
	Category            string
	ParameterID         string
	ParameterName       string
	ParameterUnit       string
	Writable            *bool
	ReadingTimestamp    string
	Value               string
	StrVal              string
	SmartHomeCategories string
	MinValue            *float64
	MaxValue            *float64
	StepValue           *float64
	EnumValues          string
	ScaleValue          string
	ZoneID              string
	RawJSON             string
}

// OpenSQLite opens (and creates if required) the sqlite database and ensures the schema is ready.
func OpenSQLite(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("ensure sqlite directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS telemetry (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id TEXT NOT NULL,
			stored_at TEXT NOT NULL,
			category TEXT,
			parameter_id TEXT,
			parameter_name TEXT,
			parameter_unit TEXT,
			writable INTEGER,
			reading_timestamp TEXT,
			value TEXT,
			str_val TEXT,
			smart_home_categories TEXT,
			min_value REAL,
			max_value REAL,
			step_value REAL,
			enum_values TEXT,
			scale_value TEXT,
			zone_id TEXT,
			raw_json TEXT NOT NULL
		);
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("ensure telemetry table: %w", err)
	}

	return &DB{sql: db}, nil
}

// Close closes the underlying database.
func (db *DB) Close() error {
	if db == nil || db.sql == nil {
		return nil
	}
	return db.sql.Close()
}

// InsertTelemetry stores a batch of telemetry records in the database.
func (db *DB) InsertTelemetry(ctx context.Context, records []TelemetryRecord) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := db.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO telemetry (
			device_id, stored_at, category, parameter_id, parameter_name,
			parameter_unit, writable, reading_timestamp, value, str_val,
			smart_home_categories, min_value, max_value, step_value,
			enum_values, scale_value, zone_id, raw_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, r := range records {
		var writable any
		if r.Writable != nil {
			if *r.Writable {
				writable = 1
			} else {
				writable = 0
			}
		} else {
			writable = nil
		}

		_, err = stmt.ExecContext(
			ctx,
			r.DeviceID,
			r.StoredAt.UTC().Format(time.RFC3339Nano),
			r.Category,
			r.ParameterID,
			r.ParameterName,
			r.ParameterUnit,
			writable,
			r.ReadingTimestamp,
			r.Value,
			r.StrVal,
			r.SmartHomeCategories,
			nullableFloat64(r.MinValue),
			nullableFloat64(r.MaxValue),
			nullableFloat64(r.StepValue),
			r.EnumValues,
			r.ScaleValue,
			r.ZoneID,
			r.RawJSON,
		)
		if err != nil {
			return fmt.Errorf("insert telemetry: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit telemetry: %w", err)
	}
	return nil
}

func nullableFloat64(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}
