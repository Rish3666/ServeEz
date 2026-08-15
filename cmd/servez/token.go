package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Rish3666/ServeEz/internal/config"
)

func init() {
	registerCommand(Command{
		Name:  "token",
		Usage: "servez token [--dir=.servez] — print the join token",
		Run:   runToken,
	})
}

func runToken(args []string) error {
	fs := flag.NewFlagSet("token", flag.ContinueOnError)
	dir := fs.String("dir", ".servez", "config dir")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := loadControlConfig(*dir)
	if err != nil {
		return err
	}
	tok, err := readToken(cfg)
	if err != nil {
		return err
	}
	fmt.Println(tok)
	return nil
}

func readToken(cfg *config.Config) (string, error) {
	if cfg.JoinToken != "" {
		return cfg.JoinToken, nil
	}
	if cfg.TokenFile != "" {
		b, err := os.ReadFile(cfg.TokenFile)
		if err == nil && len(b) > 0 {
			return string(b), nil
		}
	}
	return "", fmt.Errorf("no join token found; run `servez init` first")
}
