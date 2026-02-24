package billing

import "testing"

func TestGetPlanByTier(t *testing.T) {
	catalog := NewCatalog(PriceConfig{})

	plan, ok := catalog.GetPlanByTier("growth")
	if !ok {
		t.Fatal("expected growth plan to exist")
	}
	if plan.Tier != TierGrowth {
		t.Fatalf("expected growth tier, got %s", plan.Tier)
	}

	free, ok := catalog.GetPlanByTier("FREE")
	if !ok {
		t.Fatal("expected free plan to exist")
	}
	if free.Tier != TierFree {
		t.Fatalf("expected free tier, got %s", free.Tier)
	}
}

func TestGetLimits(t *testing.T) {
	catalog := NewCatalog(PriceConfig{})

	freeLimits, ok := catalog.GetLimits("free")
	if !ok {
		t.Fatal("expected free limits")
	}
	if freeLimits.CustomerLimit != 10 {
		t.Fatalf("expected free customer limit 10, got %d", freeLimits.CustomerLimit)
	}
	if freeLimits.IntegrationLimit != 1 {
		t.Fatalf("expected free integration limit 1, got %d", freeLimits.IntegrationLimit)
	}

	scaleLimits, ok := catalog.GetLimits("scale")
	if !ok {
		t.Fatal("expected scale limits")
	}
	if scaleLimits.CustomerLimit != Unlimited {
		t.Fatalf("expected scale customer limit unlimited, got %d", scaleLimits.CustomerLimit)
	}
}

func TestPriceMappingMonthlyAnnual(t *testing.T) {
	catalog := NewCatalog(PriceConfig{
		GrowthMonthly: "price_growth_monthly",
		GrowthAnnual:  "price_growth_annual",
		ScaleMonthly:  "price_scale_monthly",
		ScaleAnnual:   "price_scale_annual",
	})

	monthly, err := catalog.GetPriceID("growth", false)
	if err != nil {
		t.Fatalf("expected growth monthly price id, got error: %v", err)
	}
	if monthly != "price_growth_monthly" {
		t.Fatalf("expected growth monthly id, got %s", monthly)
	}

	annual, err := catalog.GetPriceID("growth", true)
	if err != nil {
		t.Fatalf("expected growth annual price id, got error: %v", err)
	}
	if annual != "price_growth_annual" {
		t.Fatalf("expected growth annual id, got %s", annual)
	}

	tier, cycle, ok := catalog.ResolveTierAndCycleByPriceID("price_scale_annual")
	if !ok {
		t.Fatal("expected price id mapping for scale annual")
	}
	if tier != TierScale {
		t.Fatalf("expected scale tier, got %s", tier)
	}
	if cycle != BillingCycleAnnual {
		t.Fatalf("expected annual cycle, got %s", cycle)
	}
}
