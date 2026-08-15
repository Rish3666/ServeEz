package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Rish3666/ServeEz/internal/agent"
	"github.com/Rish3666/ServeEz/internal/container"
	"github.com/Rish3666/ServeEz/internal/metrics"
)

func main() {
	var controlPlane string
	var token string
	var nodeID string
	var version string
	var runtimeName string
	var provider string
	var dockerHost string
	var dataDir string

	flag.StringVar(&controlPlane, "control-plane", "", "control plane base URL")
	flag.StringVar(&token, "token", "", "join token")
	flag.StringVar(&nodeID, "node-id", "", "stable node identifier")
	flag.StringVar(&version, "version", "dev", "agent version")
	flag.StringVar(&runtimeName, "runtime", "docker", "node runtime")
	flag.StringVar(&provider, "provider", "local", "node provider")
	flag.StringVar(&dockerHost, "docker-host", "", "docker daemon host")
	flag.StringVar(&dataDir, "data-dir", "", "agent state directory")
	flag.Parse()

	if controlPlane == "" || token == "" || nodeID == "" {
		fmt.Fprintln(os.Stderr, "--control-plane, --token, and --node-id are required")
		os.Exit(2)
	}

	collector := metrics.NewCollector("/")
	buffer := metrics.NewBuffer(5, 5_000_000_000)
	manager, err := container.NewDockerManager(container.DockerConfig{Host: dockerHost})
	if err != nil {
		log.Fatal(err)
	}

	cfg := agent.Config{
		ControlPlaneURL: controlPlane,
		Token:           token,
		NodeID:          nodeID,
		Version:         version,
		Runtime:         runtimeName,
		Provider:        provider,
		DataDir:         dataDir,
	}

	ag := agent.New(cfg, collector, buffer, manager, log.New(os.Stdout, "", log.LstdFlags))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := ag.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
