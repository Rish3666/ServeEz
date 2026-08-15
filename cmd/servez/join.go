package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rish3666/ServeEz/internal/agentnet"
	"github.com/Rish3666/ServeEz/internal/api"
	"github.com/Rish3666/ServeEz/internal/metrics"
)

func init() {
	registerCommand(Command{
		Name:  "join",
		Usage: "servez join <url> --token=<tok> [--provider=local] [--runtime=docker]",
		Run:   runJoin,
	})
}

func runJoin(args []string) error {
	if len(args) == 0 {
		return errors.New("join requires a control-plane URL")
	}

	baseURL := args[0]
	fs := flag.NewFlagSet("join", flag.ContinueOnError)
	token := fs.String("token", "", "join token")
	provider := fs.String("provider", "local", "node provider")
	runtimeName := fs.String("runtime", "docker", "node runtime")
	nodeIDFlag := fs.String("node-id", "", "node identifier")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *token == "" {
		return errors.New("--token is required")
	}

	nodeID := *nodeIDFlag
	if nodeID == "" {
		var err error
		nodeID, err = stableNodeID()
		if err != nil {
			return err
		}
	}

	capacity, err := metrics.DetectCapacity("/")
	if err != nil {
		return err
	}

	client, err := agentnet.New(baseURL, nil)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := client.Register(ctx, api.RegisterRequest{
		NodeID:   nodeID,
		Token:    *token,
		Runtime:  *runtimeName,
		Provider: *provider,
		Capacity: capacity,
	})
	if err != nil {
		var httpErr *agentnet.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == 401 {
			return errors.New("invalid or expired join token")
		}
		return err
	}
	if !resp.Approved {
		if resp.Reason != "" {
			return errors.New(resp.Reason)
		}
		return errors.New("join request rejected")
	}

	if err := persistNodeID(nodeID); err != nil {
		return err
	}

	fmt.Println("✓ Node registered")
	fmt.Println("✓ Agent started")
	fmt.Println("✓ Ready")
	return nil
}

func stableNodeID() (string, error) {
	if id, err := readPersistedNodeID(); err == nil && id != "" {
		return id, nil
	}
	host, err := os.Hostname()
	if err == nil {
		host = strings.TrimSpace(host)
	}
	if host == "" {
		host = randomHexID()
	}
	if err := persistNodeID(host); err != nil {
		return "", err
	}
	return host, nil
}

func nodeIDPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".servez", "node-id"), nil
}

func readPersistedNodeID() (string, error) {
	path, err := nodeIDPath()
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func persistNodeID(id string) error {
	if id == "" {
		return nil
	}
	path, err := nodeIDPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(id+"\n"), 0o600)
}

func randomHexID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "node-" + hex.EncodeToString(b[:])
}
