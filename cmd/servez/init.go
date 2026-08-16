package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rish3666/ServeEz/internal/config"
)

func init() {
	registerCommand(Command{
		Name:  "init",
		Usage: "servez init [--addr=:8443] [--dir=.servez] — write control-plane config + token",
		Run:   runInit,
	})
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	addr := fs.String("addr", ":8443", "control plane listen address")
	dir := fs.String("dir", ".servez", "config directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := os.MkdirAll(*dir, 0o755); err != nil {
		return err
	}

	cfg := config.Default()
	cfg.ListenAddr = *addr
	cfg.StatePath = filepath.Join(*dir, "state.db")
	cfg.AuditPath = filepath.Join(*dir, "audit.db")
	cfg.HistoryPath = filepath.Join(*dir, "history.db")
	cfg.TokenFile = filepath.Join(*dir, "join-token.txt")

	cfgPath := filepath.Join(*dir, "control.json")
	if err := cfg.Save(cfgPath); err != nil {
		return err
	}
	fmt.Printf("✓ wrote %s\n", cfgPath)
	fmt.Printf("  run:  servez-control --config %s\n", cfgPath)
	fmt.Printf("  join: servez join http://localhost%s --token=$(cat %s)\n", *addr, cfg.TokenFile)
	return nil
}

// loadControlConfig reads a control-plane config from a directory.
func loadControlConfig(dir string) (*config.Config, error) {
	return config.Load(filepath.Join(dir, "control.json"))
}

// readStringFlag parses a "key=value" arg.
func readStringFlag(args []string, name, def string) string {
	out := def
	for _, a := range args {
		if len(a) > len(name)+1 && a[:len(name)+1] == name+"=" {
			out = a[len(name)+1:]
		}
	}
	return out
}

func mustJSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
