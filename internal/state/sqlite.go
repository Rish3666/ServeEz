package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
	_ "modernc.org/sqlite"
)

// sqliteStore persists objects in an embedded SQLite database.
// Object payloads are stored as JSON blobs keyed by kind/namespace/name.
type sqliteStore struct {
	db  *sql.DB
	reg *Registry
}

// OpenSQLite opens (creating if needed) an object store at path.
func OpenSQLite(path string) (Store, error) {
	return OpenSQLiteWithRegistry(path, nil)
}

// OpenSQLiteWithRegistry opens an object store that decodes typed Spec/Status
// fields using reg.
func OpenSQLiteWithRegistry(path string, reg *Registry) (Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // modernc sqlite: single writer
	if _, err := db.Exec(schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return &sqliteStore{db: db, reg: reg}, nil
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS objects (
	kind            TEXT NOT NULL,
	namespace       TEXT NOT NULL DEFAULT 'default',
	name            TEXT NOT NULL,
	schema_version  TEXT NOT NULL,
	resource_version TEXT NOT NULL,
	payload         TEXT NOT NULL,
	created_at      TEXT NOT NULL,
	updated_at      TEXT NOT NULL,
	PRIMARY KEY (kind, namespace, name)
);
CREATE INDEX IF NOT EXISTS idx_objects_kind ON objects(kind);
`

func (s *sqliteStore) Create(ctx context.Context, obj *api.Object) (api.ResourceVersion, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	ns := orDefault(obj.Namespace)
	now := time.Now().UTC()
	obj.CreatedAt = now
	obj.UpdatedAt = now
	obj.ResourceVersion = newResourceVersion()

	payload, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO objects (kind, namespace, name, schema_version, resource_version, payload, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		obj.Kind, ns, obj.Name, obj.SchemaVersion, string(obj.ResourceVersion), string(payload),
		now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return "", fmt.Errorf("insert %s/%s: %w", obj.Kind, obj.Name, err)
	}
	return obj.ResourceVersion, nil
}

func (s *sqliteStore) Get(ctx context.Context, kind, namespace, name string) (*api.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT payload FROM objects WHERE kind=? AND namespace=? AND name=?`,
		kind, orDefault(namespace), name)
	var payload string
	if err := row.Scan(&payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var obj api.Object
	if err := json.Unmarshal([]byte(payload), &obj); err != nil {
		return nil, err
	}
	return s.decodeObject(&obj), nil
}

func (s *sqliteStore) List(ctx context.Context, kind, namespace string) ([]*api.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	query := `SELECT payload FROM objects`
	var args []any
	if kind != "" {
		query += ` WHERE kind=?`
		args = append(args, kind)
		if namespace != "" {
			query += ` AND namespace=?`
			args = append(args, namespace)
		}
	} else if namespace != "" {
		query += ` WHERE namespace=?`
		args = append(args, namespace)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*api.Object
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var obj api.Object
		if err := json.Unmarshal([]byte(payload), &obj); err != nil {
			return nil, err
		}
		out = append(out, s.decodeObject(&obj))
	}
	return out, rows.Err()
}

func (s *sqliteStore) Update(ctx context.Context, obj *api.Object) (api.ResourceVersion, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	ns := orDefault(obj.Namespace)
	payload, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()

	var existingPayload string
	err = s.db.QueryRowContext(ctx,
		`SELECT payload FROM objects WHERE kind=? AND namespace=? AND name=?`,
		obj.Kind, ns, obj.Name).Scan(&existingPayload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	var existing api.Object
	if err := json.Unmarshal([]byte(existingPayload), &existing); err != nil {
		return "", err
	}
	if existing.ResourceVersion != obj.ResourceVersion {
		return "", ErrConflict
	}

	obj.CreatedAt = existing.CreatedAt
	obj.UpdatedAt = now
	obj.ResourceVersion = newResourceVersion()

	_, err = s.db.ExecContext(ctx,
		`UPDATE objects SET payload=?, schema_version=?, resource_version=?, updated_at=? WHERE kind=? AND namespace=? AND name=?`,
		string(payload), obj.SchemaVersion, string(obj.ResourceVersion), now.Format(time.RFC3339Nano),
		obj.Kind, ns, obj.Name)
	if err != nil {
		return "", fmt.Errorf("update %s/%s: %w", obj.Kind, obj.Name, err)
	}
	return obj.ResourceVersion, nil
}

func (s *sqliteStore) Delete(ctx context.Context, kind, namespace, name string, version api.ResourceVersion) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var existing api.Object
	err := s.db.QueryRowContext(ctx,
		`SELECT payload FROM objects WHERE kind=? AND namespace=? AND name=?`,
		kind, orDefault(namespace), name).Scan(&existing)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if version != "" && existing.ResourceVersion != version {
		return ErrConflict
	}
	_, err = s.db.ExecContext(ctx,
		`DELETE FROM objects WHERE kind=? AND namespace=? AND name=?`,
		kind, orDefault(namespace), name)
	return err
}

func (s *sqliteStore) Watch(ctx context.Context) (<-chan WatchEvent, error) {
	// SQLite has no native change feed for arbitrary writes. For the embedded
	// backend we poll for updated_at / resource_version changes as a minimal
	// watch. The Postgres/etcd backends will use LISTEN-NOTIFY / native watches.
	ch := make(chan WatchEvent, 256)
	go func() {
		defer close(ch)
		var lastUpdated string
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
				rows, err := s.db.QueryContext(ctx,
					`SELECT payload FROM objects WHERE updated_at > ? ORDER BY updated_at`, lastUpdated)
				if err != nil {
					continue
				}
				var updates []*api.Object
				for rows.Next() {
					var payload string
					if err := rows.Scan(&payload); err != nil {
						break
					}
					var obj api.Object
					if err := json.Unmarshal([]byte(payload), &obj); err != nil {
						continue
					}
					updates = append(updates, &obj)
					if obj.UpdatedAt.Format(time.RFC3339Nano) > lastUpdated {
						lastUpdated = obj.UpdatedAt.Format(time.RFC3339Nano)
					}
				}
				rows.Close()
				for _, obj := range updates {
					select {
					case ch <- WatchEvent{Kind: obj.Kind, Object: s.decodeObject(obj), Action: "update", At: obj.UpdatedAt}:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return ch, nil
}

func (s *sqliteStore) Close() error { return s.db.Close() }

// decodeObject re-encodes Spec/Status into typed structs when a registry is set.
func (s *sqliteStore) decodeObject(obj *api.Object) *api.Object {
	if s.reg == nil {
		return obj
	}
	if spec, err := s.reg.DecodeSpec(obj.Kind, obj.Spec); err == nil {
		obj.Spec = spec
	}
	if status, err := s.reg.DecodeStatus(obj.Kind, obj.Status); err == nil {
		obj.Status = status
	}
	return obj
}

func orDefault(ns string) string {
	if ns == "" {
		return "default"
	}
	return ns
}
