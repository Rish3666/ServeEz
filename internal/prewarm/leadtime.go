// Package prewarm estimates the time needed to make additional workload
// replicas ready on a node.
package prewarm

import (
	"strings"
	"time"
)

// LeadTime returns a conservative MVP estimate for the requested runtime and
// image reference. The shape leaves room for replacing estimates with runtime
// measurements later.
func LeadTime(runtime, image string) time.Duration {
	if strings.EqualFold(runtime, "docker") || runtime == "" {
		pull := 15 * time.Second
		switch {
		case strings.Contains(image, "cuda"), strings.Contains(image, "gpu"):
			pull = 45 * time.Second
		case strings.Contains(image, "large"), strings.Contains(image, "tensorflow"), strings.Contains(image, "pytorch"):
			pull = 30 * time.Second
		}
		return pull + 5*time.Second
	}
	return 20 * time.Second
}
