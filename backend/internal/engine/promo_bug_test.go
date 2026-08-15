package engine

import (
	"promo-engine/internal/models"
	"testing"
	"time"
)

func TestCalculateFullReductionDiscount_AppliesAtExactThreshold(t *testing.T) {
	e := &PromoEngine{}
	result := &models.CalculationResult{
		Items: []models.CalculatedItem{{SKUId: 1, PayPrice: 100, Quantity: 1}},
	}
	rule := &models.PromoRule{
		Scope: models.PromoScope{Type: "all"},
		Config: models.JSONB{
			"tiers": []map[string]interface{}{
				{"threshold": 100.0, "discount": 20.0},
			},
		},
	}
	got := e.calculateFullReductionDiscount(rule, result)
	if got != 20 {
		t.Fatalf("exact threshold discount=%v, want 20", got)
	}
}

func TestCalculateNthItemDiscount_SecondItemOnly(t *testing.T) {
	e := &PromoEngine{}
	result := &models.CalculationResult{
		Items: []models.CalculatedItem{{SKUId: 1, PayPrice: 80, Quantity: 1}},
	}
	rule := &models.PromoRule{
		Scope: models.PromoScope{Type: "all"},
		Config: models.JSONB{
			"nth_item":      2,
			"discount_rate": 0.0,
			"free":          true,
		},
	}
	got := e.calculateNthItemDiscount(rule, result)
	if got != 0 {
		t.Fatalf("single item nth=2 discount=%v, want 0", got)
	}
}

func TestHasTimeOverlap_TouchingRangesOverlap(t *testing.T) {
	e := &PromoEngine{}
	t1 := &models.TimeCondition{
		StartTime: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	t2 := &models.TimeCondition{
		StartTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
	}
	if !e.hasTimeOverlap(t1, t2) {
		t.Fatal("ranges that touch at the boundary should overlap")
	}
}

func TestCalculateDiscountDiscount_BadRateTypeNoDiscount(t *testing.T) {
	e := &PromoEngine{}
	result := &models.CalculationResult{
		Items: []models.CalculatedItem{{SKUId: 1, PayPrice: 100, Quantity: 1}},
	}
	rule := &models.PromoRule{
		Scope:  models.PromoScope{Type: "all"},
		Config: models.JSONB{"discount_rate": "0.8"},
	}
	got := e.calculateDiscountDiscount(rule, result)
	if got != 0 {
		t.Fatalf("bad discount_rate type discount=%v, want 0", got)
	}
}

func TestHasScopeOverlap_AllOnSecondScope(t *testing.T) {
	e := &PromoEngine{}
	s1 := &models.PromoScope{Type: "sku", SKUIds: []int64{1}}
	s2 := &models.PromoScope{Type: "all"}
	if !e.hasScopeOverlap(s1, s2) {
		t.Fatal("sku scope vs all scope should overlap")
	}
}
