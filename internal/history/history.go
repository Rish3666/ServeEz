// Package history implements an append-only SQLite time-series store for the
// predictor engine. It records utilization samples per series (node or
// workload) from the agent's periodic reports and serves recent windows to the
// forecast model (internal/predictor). Design: AI Integration/01.
package history

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Sample is a single recorded utilization reading.
type Sample struct {
	At       time.Time
	CPU      float64 // percent 0-100
	MemPct   float64
	MemBytes uint64
}

// Store is the time-series persistence.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the time-series store at path.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open history sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init history schema: %w", err)
	}
	return &Store{db: db}, nil
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS samples (
	series   TEXT NOT NULL,
	ts       TEXT NOT NULL,
	cpu      REAL NOT NULL,
	mem_pct  REAL NOT NULL DEFAULT 0,
	mem_bytes INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_samples_series_ts ON samples(series, ts);
`

// Record appends a sample for a series. series is e.g. "node:node-1" or
// "workload:web".
func (s *Store) Record(ctx context.Context, series string, smp Sample) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO samples (series, ts, cpu, mem_pct, mem_bytes) VALUES (?, ?, ?, ?, ?)`,
		series, smp.At.UTC().Format(time.RFC3339Nano), smp.CPU, smp.MemPct, smp.MemBytes)
	if err != nil {
		return fmt.Errorf("record sample: %w", err)
	}
	return nil
}

// Recent returns the last n samples for a series, oldest first.
func (s *Store) Recent(ctx context.Context, series string, n int) ([]Sample, error) {
	if n <= 0 {
		n = 1
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT ts, cpu, mem_pct, mem_bytes FROM samples WHERE series=? ORDER BY ts DESC LIMIT ?`,
		series, n)
	if err != nil {
		return nil, fmt.Errorf("query samples: %w", err)
	}
	defer rows.Close()
	var out []Sample
	for rows.Next() {
		var ts string
		var smp Sample
		if err := rows.Scan(&ts, &smp.CPU, &smp.MemPct, &smp.MemBytes); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			return nil, fmt.Errorf("parse sample ts: %w", err)
		}
		smp.At = t
		out = append(out, smp)
	}
	// Reverse to oldest-first.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, rows.Err()
}

// Count returns the number of samples recorded for a series.
func (s *Store) Count(ctx context.Context, series string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM samples WHERE series=?`, series).Scan(&n)
	return n, err
}

// Prune deletes samples older than the cutoff, keeping the store bounded.
func (s *Store) Prune(ctx context.Context, cutoff time.Time) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM samples WHERE ts < ?`, cutoff.UTC().Format(time.RFC3339Nano))
	return err
}

// Close closes the store.
func (s *Store) Close() error { return s.db.Close() }