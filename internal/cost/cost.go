// Package cost implements the multi-cloud pricing comparison engine
// (Core Features/03, AI Integration/04). It compares AWS/Azure/GCP instance
// pricing for a given workload shape and recommends the cheapest option.
//
// MVP: offline baseline catalog (2026 typical pricing per region). Live
// scraping of spot APIs is a later sprint; the interface (api.CostComparer)
// is unchanged either way.
package cost

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/Rish3666/ServeEz/internal/api"
)

// HoursPerMonth is the standard cloud billing month.
const HoursPerMonth = 730.0

// InstanceOffer is one purchasable instance shape at baseline pricing.
type InstanceOffer struct {
	Provider      string // "aws" | "azure" | "gcp"
	InstanceType  string // e.g. "m5.xlarge"
	Region        string // e.g. "us-east-1"
	VCPU          int
	MemGB         float64
	OnDemandPerHr float64 // $/hr on-demand
	SpotPerHr     float64 // $/hr spot (0 if not offered)
	Available     bool
}

// PriceCatalog is an offline collection of provider instance offers.
type PriceCatalog struct {
	Offers []InstanceOffer
}

// Engine implements api.CostComparer over a static price catalog.
type Engine struct {
	catalog PriceCatalog
}

// New builds an engine with the default baseline catalog.
func New() *Engine {
	return NewWithCatalog(PriceCatalog{Offers: defaultCatalog()})
}

// NewWithCatalog builds an engine over a caller-provided catalog.
func NewWithCatalog(catalog PriceCatalog) *Engine {
	offers := append([]InstanceOffer(nil), catalog.Offers...)
	return &Engine{catalog: PriceCatalog{Offers: offers}}
}

// Compare prices a workload shape across all providers (fast tier, <10ms).
func (e *Engine) Compare(ctx context.Context, req api.CostCompareRequest) (api.CostReport, error) {
	if err := validate(req); err != nil {
		return api.CostReport{}, err
	}
	runtime := req.RuntimePct
	if runtime <= 0 {
		runtime = 100
	}
	region := req.Region
	if region == "" {
		region = "us-east-1"
	}

	// Best (cheapest) matching offer per provider.
	byProvider := map[string]*api.CostRecommendation{}
	for _, prov := range []string{"aws", "azure", "gcp"} {
		rec, ok := bestOffer(e.catalog.Offers, prov, region, req.VCPU, req.MemGB, runtime)
		if !ok {
			continue
		}
		byProvider[prov] = rec
	}
	if len(byProvider) == 0 {
		return api.CostReport{}, fmt.Errorf("no provider offers a matching instance for %d vCPU / %d GB in %s", req.VCPU, req.MemGB, region)
	}

	providers := make([]api.CostRecommendation, 0, len(byProvider))
	for _, rec := range byProvider {
		providers = append(providers, *rec)
	}
	sort.Slice(providers, func(i, j int) bool { return providers[i].EstMonthly < providers[j].EstMonthly })

	best := providers[0]
	worst := providers[len(providers)-1].EstMonthly
	savings := 0.0
	if worst > 0 {
		savings = (worst - best.EstMonthly) / worst * 100
	}
	return api.CostReport{
		Request:             req,
		Best:                &best,
		Providers:           providers,
		PotentialSavingsPct: savings,
	}, nil
}

func validate(req api.CostCompareRequest) error {
	if req.VCPU <= 0 {
		return errors.New("vcpu must be > 0")
	}
	if req.MemGB <= 0 {
		return errors.New("mem_gb must be > 0")
	}
	if req.RuntimePct < 0 || req.RuntimePct > 100 {
		return errors.New("runtime_pct must be 0-100")
	}
	return nil
}

// bestOffer returns the cheapest offer from a provider that satisfies the
// shape, or (nil,false) if none matches. If the requested region has no
// matching offer, it falls back to the provider's default region so every
// provider is always comparable.
func bestOffer(offers []InstanceOffer, provider, region string, vcpu, memGB int, runtime float64) (*api.CostRecommendation, bool) {
	var best *InstanceOffer
	bestOnDemand := 0.0
	foundRegion := false
	for i := range offers {
		o := &offers[i]
		if o.Provider != provider {
			continue
		}
		if !o.Available {
			continue
		}
		if o.Region != region {
			continue
		}
		foundRegion = true
		if o.VCPU < vcpu || o.MemGB < float64(memGB) {
			continue
		}
		if best == nil || o.OnDemandPerHr < bestOnDemand {
			best = o
			bestOnDemand = o.OnDemandPerHr
		}
	}
	if best == nil {
		// No offer in the requested region satisfied the shape. Try any other
		// region for this provider (fallback so every provider is compared).
		if !foundRegion {
			for i := range offers {
				o := &offers[i]
				if o.Provider != provider || !o.Available || o.VCPU < vcpu || o.MemGB < float64(memGB) {
					continue
				}
				if best == nil || o.OnDemandPerHr < bestOnDemand {
					best = o
					bestOnDemand = o.OnDemandPerHr
				}
			}
		}
	}
	if best == nil {
		return nil, false
	}

	onDemandMo := best.OnDemandPerHr * HoursPerMonth * runtime / 100
	spotMo := best.SpotPerHr * HoursPerMonth * runtime / 100
	recommended := "on_demand"
	est := onDemandMo
	// Prefer spot only when meaningfully cheaper and offered.
	if best.SpotPerHr > 0 && spotMo < onDemandMo*0.85 {
		recommended = "spot"
		est = spotMo
	}
	return &api.CostRecommendation{
		Provider:      best.Provider,
		InstanceType:  best.InstanceType,
		Region:        best.Region,
		VCPU:          best.VCPU,
		MemGB:         int(best.MemGB),
		OnDemandPerMo: onDemandMo,
		SpotPerMo:     spotMo,
		Recommended:   recommended,
		EstMonthly:    est,
	}, true
}

// defaultCatalog returns the baseline 2026 pricing table (typical, per
// provider for a few common regions; values are representative).
func defaultCatalog() []InstanceOffer {
	o := []InstanceOffer{
		// ===== AWS (us-east-1) =====
		{"aws", "t3.micro", "us-east-1", 2, 1, 0.0104, 0.0031, true},
		{"aws", "t3.small", "us-east-1", 2, 2, 0.0208, 0.0062, true},
		{"aws", "t3.medium", "us-east-1", 2, 4, 0.0416, 0.0125, true},
		{"aws", "t3.large", "us-east-1", 2, 8, 0.0832, 0.0250, true},
		{"aws", "t3.xlarge", "us-east-1", 4, 16, 0.1664, 0.0499, true},
		{"aws", "t3.2xlarge", "us-east-1", 8, 32, 0.3328, 0.0998, true},
		{"aws", "m5.large", "us-east-1", 2, 8, 0.096, 0.0288, true},
		{"aws", "m5.xlarge", "us-east-1", 4, 16, 0.192, 0.0576, true},
		{"aws", "m5.2xlarge", "us-east-1", 8, 32, 0.384, 0.1152, true},
		{"aws", "c5.large", "us-east-1", 2, 4, 0.085, 0.0255, true},
		{"aws", "c5.xlarge", "us-east-1", 4, 8, 0.17, 0.0510, true},
		{"aws", "r5.large", "us-east-1", 2, 16, 0.126, 0.0378, true},
		{"aws", "r5.xlarge", "us-east-1", 4, 32, 0.252, 0.0756, true},

		// ===== Azure (eastus) =====
		{"azure", "Standard_B2s", "eastus", 2, 4, 0.0416, 0.0125, true},
		{"azure", "Standard_B2ms", "eastus", 2, 8, 0.0832, 0.0250, true},
		{"azure", "Standard_B4ms", "eastus", 4, 16, 0.1664, 0.0499, true},
		{"azure", "Standard_D2s_v3", "eastus", 2, 8, 0.096, 0.0288, true},
		{"azure", "Standard_D4s_v3", "eastus", 4, 16, 0.192, 0.0576, true},
		{"azure", "Standard_D2s_v5", "eastus", 2, 8, 0.085, 0.0255, true},
		{"azure", "Standard_D4s_v5", "eastus", 4, 16, 0.17, 0.0510, true},
		{"azure", "Standard_E2s_v5", "eastus", 2, 16, 0.12, 0.0360, true},
		{"azure", "Standard_E4s_v5", "eastus", 4, 32, 0.24, 0.0720, true},
		{"azure", "Standard_F2s_v2", "eastus", 2, 4, 0.084, 0.0252, true},
		{"azure", "Standard_F4s_v2", "eastus", 4, 8, 0.168, 0.0504, true},

		// ===== GCP (us-east1) =====
		{"gcp", "e2-small", "us-east1", 2, 2, 0.032, 0.0096, true},
		{"gcp", "e2-medium", "us-east1", 2, 4, 0.048, 0.0144, true},
		{"gcp", "e2-standard-2", "us-east1", 2, 8, 0.067, 0.0201, true},
		{"gcp", "e2-standard-4", "us-east1", 4, 16, 0.134, 0.0402, true},
		{"gcp", "e2-standard-8", "us-east1", 8, 32, 0.268, 0.0804, true},
		{"gcp", "n1-standard-2", "us-east1", 2, 7.5, 0.095, 0.0285, true},
		{"gcp", "n1-standard-4", "us-east1", 4, 15, 0.19, 0.0570, true},
		{"gcp", "n2-standard-2", "us-east1", 2, 8, 0.097, 0.0291, true},
		{"gcp", "n2-standard-4", "us-east1", 4, 16, 0.195, 0.0585, true},
		{"gcp", "c3-standard-4", "us-east1", 4, 16, 0.207, 0.0621, true},
	}
	return o
}
