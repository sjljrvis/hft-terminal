package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"hft/pkg/types"
)

// TickStore provides basic CRUD helpers for ticks stored in SQLite.
type TickStore struct {
	db *sql.DB
}

// NewTickStore opens (or creates) the SQLite database at the given path and ensures schema exists.
// Prefer using NewDB for multi-model access.
func NewTickStore(dbPath string) (*TickStore, error) {
	if dbPath == "" {
		return nil, errors.New("db path is required")
	}
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, fmt.Errorf("resolve db path: %w", err)
	}

	db, err := sql.Open("sqlite", absPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	s := &TickStore{db: db}
	if err := s.ensureSchema(); err != nil {
		return nil, err
	}
	return s, nil
}

// Close releases the underlying database connection.
func (s *TickStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *TickStore) ensureSchema() error {
	const ddl = `
CREATE TABLE IF NOT EXISTS ticks (
  id INTEGER PRIMARY KEY,
  timestamp TEXT NOT NULL,
  symbol TEXT NOT NULL,
  tf TEXT NOT NULL,
  open REAL NOT NULL,
  high REAL NOT NULL,
  low REAL NOT NULL,
  close REAL NOT NULL,
  time INTEGER NOT NULL,
  volume REAL NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ticks_time ON ticks(time);
CREATE INDEX IF NOT EXISTS idx_ticks_symbol_tf_time ON ticks(symbol, tf, time);
`
	if _, err := s.db.Exec(ddl); err != nil {
		return fmt.Errorf("ensure ticks schema: %w", err)
	}
	return nil
}

// FindAll returns all ticks ordered by time asc.
func (s *TickStore) FindAll(ctx context.Context) ([]types.Tick, error) {
	return s.ListTicksFiltered(ctx, "", "", 0, "", "")
}

// SeedSample inserts a single sample tick if the table is empty.
func (s *TickStore) SeedSample(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ticks`).Scan(&count); err != nil {
		return fmt.Errorf("count ticks: %w", err)
	}
	if count > 0 {
		return nil
	}
	sample := types.Tick{
		ID:        2866124,
		Timestamp: time.Date(2025, 8, 19, 13, 43, 0, 0, time.FixedZone("IST", 19800)),
		Symbol:    "nifty",
		TF:        "1",
		Open:      24964.7,
		High:      24973.35,
		Low:       24964.7,
		Close:     24967.6,
		Time:      1755591180,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_, err := s.InsertTicks(ctx, []types.Tick{sample})
	return err
}

// InsertTicks bulk inserts ticks.
func (s *TickStore) InsertTicks(ctx context.Context, ticks []types.Tick) (int64, error) {
	if len(ticks) == 0 {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO ticks (id, timestamp, symbol, tf, open, high, low, close, time, volume, created_at, updated_at)
VALUES (COALESCE(?, NULL), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`)
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	var inserted int64
	for _, t := range ticks {
		created := t.CreatedAt
		updated := t.UpdatedAt
		if created.IsZero() {
			created = time.Now().UTC()
		}
		if updated.IsZero() {
			updated = created
		}
		if _, err := stmt.ExecContext(
			ctx,
			nullID(t.ID),
			t.Timestamp.Format(time.RFC3339),
			t.Symbol,
			t.TF,
			t.Open,
			t.High,
			t.Low,
			t.Close,
			t.Time,
			t.Volume,
			created.Format(time.RFC3339Nano),
			updated.Format(time.RFC3339Nano),
		); err != nil {
			_ = tx.Rollback()
			return inserted, fmt.Errorf("insert tick: %w", err)
		}
		inserted++
	}
	if err := tx.Commit(); err != nil {
		return inserted, fmt.Errorf("commit: %w", err)
	}
	return inserted, nil
}

// ListTicks returns ticks ordered by ascending time (oldest first).
func (s *TickStore) ListTicks(ctx context.Context, limit int) ([]types.Tick, error) {
	return s.ListTicksFiltered(ctx, "", "", limit, "", "")
}

// ListTicksFiltered returns ticks, optionally filtered by symbol/tf, ordered by time ASC.
// limit <= 0 means no limit.
func (s *TickStore) ListTicksFiltered(ctx context.Context, symbol string, tf string, limit int, startDate string, endDate string) ([]types.Tick, error) {
	query := `
SELECT id, timestamp, symbol, tf, open, high, low, close, time, volume, created_at, updated_at
FROM ticks
`
	var args []any
	var conditions []string
	if symbol != "" {
		conditions = append(conditions, "symbol = ?")
		args = append(args, symbol)
	}
	if tf != "" {
		conditions = append(conditions, "tf = ?")
		args = append(args, tf)
	}
	if ts, ok := parseEpoch(startDate); ok {
		conditions = append(conditions, "time >= ?")
		args = append(args, ts)
	}
	if ts, ok := parseEpoch(endDate); ok {
		conditions = append(conditions, "time <= ?")
		args = append(args, ts)
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY time ASC"
	var rows *sql.Rows
	var err error
	if limit > 0 {
		query = query + " LIMIT ?"
		args = append(args, limit)
		rows, err = s.db.QueryContext(ctx, query, args...)
	} else {
		rows, err = s.db.QueryContext(ctx, query, args...)
	}
	if err != nil {
		return nil, fmt.Errorf("query ticks: %w", err)
	}
	defer rows.Close()

	var out []types.Tick
	for rows.Next() {
		var t types.Tick
		var ts, created, updated string
		if err := rows.Scan(
			&t.ID,
			&ts,
			&t.Symbol,
			&t.TF,
			&t.Open,
			&t.High,
			&t.Low,
			&t.Close,
			&t.Time,
			&t.Volume,
			&created,
			&updated,
		); err != nil {
			return nil, fmt.Errorf("scan tick: %w", err)
		}
		t.Timestamp, _ = time.Parse(time.RFC3339, ts)
		t.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		t.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ticks: %w", err)
	}
	return out, nil
}
