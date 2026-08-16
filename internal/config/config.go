// Package config provides typed configuration loading for ServeEz components.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config is the control-plane configuration.
type Config struct {
	// ListenAddr is the API server bind address, e.g. ":8443".
	ListenAddr string `json:"listen_addr"`

	// StatePath is the SQLite object-store path.
	StatePath string `json:"state_path"`

	// AuditPath is the SQLite audit-log path.
	AuditPath string `json:"audit_path"`

	// HistoryPath is the SQLite time-series store path (predictor input).
	HistoryPath string `json:"history_path"`

	// JoinToken is the shared registration secret. If empty, a random one is
	// generated on first boot and persisted to TokenFile.
	JoinToken string `json:"join_token"`

	// TokenFile persists the generated join token.
	TokenFile string `json:"token_file"`

	// TLS cert/key paths for the API server (mTLS in later sprints).
	TLSCertFile string `json:"tls_cert_file"`
	TLSKeyFile  string `json:"tls_key_file"`

	// WatchInterval controls the SQLite watch polling cadence.
	WatchInterval time.Duration `json:"-"`
}

// Default returns the default control-plane configuration.
func Default() *Config {
	return &Config{
		ListenAddr:    ":8443",
		StatePath:     "./servez.db",
		AuditPath:     "./servez-audit.db",
		HistoryPath:   "./servez-history.db",
		TokenFile:     "./join-token.txt",
		WatchInterval: 500 * time.Millisecond,
	}
}

// Load reads a JSON config file. Missing fields fall back to defaults.
func Load(path string) (*Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// Save writes the config to path (used by `servez init`).
func (c *Config) Save(path string) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
