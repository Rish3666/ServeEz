package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
	"github.com/Rish3666/ServeEz/internal/apiclient"
)

func init() {
	registerCommand(Command{
		Name:  "deploy",
		Usage: "servez deploy <name> --image=<img> [--replicas=N] [--cpu=0.5] [--mem-mb=512] [--type=service] [--url=...]",
		Run:   runDeploy,
	})
	registerCommand(Command{
		Name:  "scale",
		Usage: "servez scale <workload> --replicas=N [--url=...]",
		Run:   runScale,
	})
}

func runDeploy(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: servez deploy <name> --image=<img> ...")
	}
	name := args[0]
	image := readStringFlag(args, "--image", "")
	if image == "" {
		return fmt.Errorf("--image is required")
	}
	replicas := 1
	if r := readStringFlag(args, "--replicas", ""); r != "" {
		v, err := strconv.Atoi(r)
		if err != nil {
			return fmt.Errorf("invalid --replicas: %w", err)
		}
		replicas = v
	}
	url := readStringFlag(args, "--url", "http://localhost:8443")
	restart := readStringFlag(args, "--restart", "always")
	strategy := readStringFlag(args, "--strategy", "rolling")

	spec := &api.WorkloadSpec{
		Image:         image,
		Replicas:      replicas,
		Type:          readStringFlag(args, "--type", "service"),
		RestartPolicy: restart,
		Strategy:      strategy,
		Env:           map[string]string{},
	}
	if cpu := readStringFlag(args, "--cpu", ""); cpu != "" {
		v, err := strconv.ParseFloat(cpu, 64)
		if err != nil {
			return fmt.Errorf("invalid --cpu: %w", err)
		}
		if spec.Resources == nil {
			spec.Resources = &api.Resources{}
		}
		spec.Resources.CPUCores = v
	}
	if mem := readStringFlag(args, "--mem-mb", ""); mem != "" {
		v, err := strconv.Atoi(mem)
		if err != nil {
			return fmt.Errorf("invalid --mem-mb: %w", err)
		}
		if spec.Resources == nil {
			spec.Resources = &api.Resources{}
		}
		spec.Resources.MemBytes = uint64(v) << 20
	}

	c := apiclient.New(url)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := c.Deploy(ctx, name, spec); err != nil {
		return err
	}
	fmt.Printf("✓ deployed %s (%s, %d replica(s))\n", name, image, replicas)
	return nil
}

func runScale(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: servez scale <workload> --replicas=N")
	}
	target := args[0]
	replicas, err := strconv.Atoi(readStringFlag(args, "--replicas", ""))
	if err != nil {
		return fmt.Errorf("--replicas is required")
	}
	url := readStringFlag(args, "--url", "http://localhost:8443")

	c := apiclient.New(url)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	res, err := c.Execute(ctx, api.Action{
		Type:       "scale",
		Target:     "workload:" + target,
		Parameters: map[string]any{"replicas": replicas},
		Reason:     "manual scale via CLI",
		Initiator:  "human:cli",
		Confidence: 0.9,
	})
	if err != nil {
		return err
	}
	fmt.Printf("✓ scale %s → %d: %s (%s)\n", target, replicas, res.Status, res.Message)
	return nil
}
