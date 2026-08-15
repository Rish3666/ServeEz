// Package audit implements the immutable audit trail for every ServeEz action.
// Entries are append-only; there is no update or delete path. Design doc:
// AI Control/05 (Action Audit & Safety).
package audit

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
	_ "modernc.org/sqlite"
)

// ErrNotFound is returned when an entry does not exist.
var ErrNotFound = errors.New("audit entry not found")

// Log is an append-only store of audit entries.
type Log interface {
	// Append records a new entry. Returns its ID.
	Append(ctx context.Context, e *api.AuditEntry) (string, error)
	// Get fetches an entry by ID.
	Get(ctx context.Context, id string) (*api.AuditEntry, error)
	// List returns entries, newest first, optionally filtered by initiator/status.
	List(ctx context.Context, filter Filter) ([]*api.AuditEntry, error)
	// Close releases resources.
	Close() error
}

// Filter constrains List results.
type Filter struct {
	Initiator string
	Status    string
	Limit     int
}

// newID returns a short unique identifier.
func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "aud_" + hex.EncodeToString(b)
}

// SQLiteLog persists entries in an embedded SQLite database.
type SQLiteLog struct {
	db *sql.DB
}

// OpenSQLite opens (creating if needed) an audit log at path.
func OpenSQLite(path string) (*SQLiteLog, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open audit sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init audit schema: %w", err)
	}
	return &SQLiteLog{db: db}, nil
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS audit (
	id          TEXT PRIMARY KEY,
	ts          TEXT NOT NULL,
	initiator   TEXT NOT NULL,
	action_type TEXT NOT NULL,
	target      TEXT NOT NULL,
	params      TEXT,
	before      TEXT,
	after       TEXT,
	status      TEXT NOT NULL,
	confidence  REAL NOT NULL DEFAULT 0,
	duration_ms INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_audit_initiator ON audit(initiator);
CREATE INDEX IF NOT EXISTS idx_audit_status ON audit(status);
CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit(ts);
`

// Append writes an entry. Entry ID must be unique; if empty it is generated.
func (l *SQLiteLog) Append(ctx context.Context, e *api.AuditEntry) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if e.ID == "" {
		e.ID = newID()
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	params, _ := json.Marshal(e.Parameters)
	before, _ := json.Marshal(e.StateBefore)
	after, _ := json.Marshal(e.StateAfter)
	_, err := l.db.ExecContext(ctx,
		`INSERT INTO audit (id, ts, initiator, action_type, target, params, before, after, status, confidence, duration_ms)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Timestamp.Format(time.RFC3339Nano), e.Initiator, e.ActionType, e.Target,
		string(params), string(before), string(after), e.Status, e.Confidence, e.DurationMS)
	if err != nil {
		return "", fmt.Errorf("append audit: %w", err)
	}
	return e.ID, nil
}

func (l *SQLiteLog) Get(ctx context.Context, id string) (*api.AuditEntry, error) {
	row := l.db.QueryRowContext(ctx, `SELECT * FROM audit WHERE id=?`, id)
	e, err := scanEntry(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return e, err
}

func (l *SQLiteLog) List(ctx context.Context, f Filter) ([]*api.AuditEntry, error) {
	query := `SELECT * FROM audit`
	var args []any
	conds := []string{}
	if f.Initiator != "" {
		conds = append(conds, `initiator=?`)
		args = append(args, f.Initiator)
	}
	if f.Status != "" {
		conds = append(conds, `status=?`)
		args = append(args, f.Status)
	}
	for i, c := range conds {
		if i == 0 {
			query += ` WHERE ` + c
		} else {
			query += ` AND ` + c
		}
	}
	query += ` ORDER BY ts DESC`
	if f.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, f.Limit)
	}
	rows, err := l.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*api.AuditEntry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (l *SQLiteLog) Close() error { return l.db.Close() }

type scanner interface {
	Scan(dest ...any) error
}

func scanEntry(s scanner) (*api.AuditEntry, error) {
	var (
		id, ts, initiator, actionType, target, status string
		params, before, after                          string
		confidence                                     float64
		durationMS                                     int64
	)
	if err := s.Scan(&id, &ts, &initiator, &actionType, &target, &params, &before, &after, &status, &confidence, &durationMS); err != nil {
		return nil, err
	}
	t, _ := time.Parse(time.RFC3339Nano, ts)
	e := &api.AuditEntry{
		ID:         id,
		Timestamp:  t,
		Initiator:  initiator,
		ActionType: actionType,
		Target:     target,
		Status:     status,
		Confidence: confidence,
		DurationMS: durationMS,
	}
	_ = json.Unmarshal([]byte(params), &e.Parameters)
	_ = json.Unmarshal([]byte(before), &e.StateBefore)
	_ = json.Unmarshal([]byte(after), &e.StateAfter)
	return e, nil
}
