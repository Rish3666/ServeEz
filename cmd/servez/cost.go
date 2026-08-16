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
		Name:  "cost",
		Usage: "servez cost --vcpu=N --mem-gb=N [--runtime-pct=100] [--region=us-east-1] [--url=...]",
		Run:   runCost,
	})
}

func runCost(args []string) error {
	vcpu, err := strconv.Atoi(readStringFlag(args, "--vcpu", ""))
	if err != nil || vcpu <= 0 {
		return fmt.Errorf("--vcpu is required (positive int)")
	}
	memGB, err := strconv.Atoi(readStringFlag(args, "--mem-gb", ""))
	if err != nil || memGB <= 0 {
		return fmt.Errorf("--mem-gb is required (positive int)")
	}
	runtimePct := 100.0
	if v := readStringFlag(args, "--runtime-pct", ""); v != "" {
		runtimePct, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("invalid --runtime-pct: %w", err)
		}
	}
	url := readStringFlag(args, "--url", "http://localhost:8443")

	c := apiclient.New(url)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rep, err := c.CostCompare(ctx, api.CostCompareRequest{
		VCPU:       vcpu,
		MemGB:      memGB,
		RuntimePct: runtimePct,
		Region:     readStringFlag(args, "--region", "us-east-1"),
	})
	if err != nil {
		return err
	}

	fmt.Printf("Cost comparison — %d vCPU / %d GB / %.0f%% runtime\n", vcpu, memGB, runtimePct)
	for _, p := range rep.Providers {
		fmt.Printf("  %-6s %-16s on-demand $%7.2f/mo  spot $%7.2f/mo  → $%6.2f/mo (%s)\n",
			p.Provider, p.InstanceType, p.OnDemandPerMo, p.SpotPerMo, p.EstMonthly, p.Recommended)
	}
	if rep.Best != nil {
		fmt.Printf("✓ Best: %s %s @ $%.2f/mo (%s)\n", rep.Best.Provider, rep.Best.InstanceType, rep.Best.EstMonthly, rep.Best.Recommended)
		fmt.Printf("  Potential savings vs most expensive: %.1f%%\n", rep.PotentialSavingsPct)
	}
	return nil
}
