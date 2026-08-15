package main

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"time"

	"github.com/Rish3666/ServeEz/internal/apiclient"
)

func init() {
	registerCommand(Command{
		Name:  "status",
		Usage: "servez status [--url=http://localhost:8443] — show cluster state",
		Run:   runStatus,
	})
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	url := fs.String("url", "http://localhost:8443", "control plane URL")
	dir := fs.String("dir", ".servez", "config dir (for --url default)")
	jsonOut := fs.Bool("json", false, "raw JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *url == "http://localhost:8443" {
		if cfg, err := loadControlConfig(*dir); err == nil && cfg != nil {
			*url = "http://localhost" + cfg.ListenAddr
		}
	}

	c := apiclient.New(*url)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nodes, err := c.State(ctx, "Node")
	if err != nil {
		return fmt.Errorf("fetch state: %w", err)
	}
	workloads, err := c.State(ctx, "Workload")
	if err != nil {
		return fmt.Errorf("fetch workloads: %w", err)
	}

	if *jsonOut {
		fmt.Println(mustJSON(map[string]any{"nodes": nodes, "workloads": workloads}))
		return nil
	}

	fmt.Printf("Nodes (%d):\n", len(nodes))
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Name < nodes[j].Name })
	for _, n := range nodes {
		st := n.Status
		state, health := "-", 0
		if st != nil {
			state = stateOf(st)
			if m, ok := st.(map[string]any); ok {
				if h, ok := m["health_score"].(float64); ok {
					health = int(h)
				}
			}
		}
		fmt.Printf("  %-12s state=%s health=%d\n", n.Name, state, health)
	}

	fmt.Printf("\nWorkloads (%d):\n", len(workloads))
	sort.Slice(workloads, func(i, j int) bool { return workloads[i].Name < workloads[j].Name })
	for _, w := range workloads {
		ws, _ := w.Spec.(map[string]any)
		wst, _ := w.Status.(map[string]any)
		image, replicas, state, node := "-", 0, "-", "-"
		if ws != nil {
			if i, ok := ws["image"].(string); ok {
				image = i
			}
			if r, ok := ws["replicas"].(float64); ok {
				replicas = int(r)
			}
		}
		if wst != nil {
			if s, ok := wst["state"].(string); ok {
				state = s
			}
			if n, ok := wst["assigned_node"].(string); ok {
				node = n
			}
		}
		fmt.Printf("  %-12s image=%-16s replicas=%d state=%s node=%s\n", w.Name, image, replicas, state, node)
	}
	return nil
}

func stateOf(st any) string {
	m, ok := st.(map[string]any)
	if !ok {
		return "-"
	}
	if s, ok := m["state"].(string); ok {
		return s
	}
	return "-"
}
