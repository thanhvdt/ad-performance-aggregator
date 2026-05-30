package aggregate

import "testing"

func TestStatsAdd(t *testing.T) {
	var s Stats
	s.Add(100, 10, 500, 2) // impressions, clicks, spendCents, conversions
	s.Add(50, 5, 250, 1)
	want := Stats{Impressions: 150, Clicks: 15, Conversions: 3, SpendCents: 750}
	if s != want {
		t.Errorf("Add = %+v, want %+v", s, want)
	}
}

func TestMerge(t *testing.T) {
	dst := map[string]*Stats{
		"CMP001": {Impressions: 100, Clicks: 10, Conversions: 1, SpendCents: 500},
	}
	src := map[string]*Stats{
		"CMP001": {Impressions: 50, Clicks: 5, Conversions: 1, SpendCents: 250}, // folds into existing
		"CMP002": {Impressions: 20, Clicks: 2, Conversions: 0, SpendCents: 100}, // brand-new campaign
	}
	Merge(dst, src)

	if len(dst) != 2 {
		t.Fatalf("dst has %d campaigns, want 2", len(dst))
	}
	if got, want := *dst["CMP001"], (Stats{Impressions: 150, Clicks: 15, Conversions: 2, SpendCents: 750}); got != want {
		t.Errorf("merged CMP001 = %+v, want %+v", got, want)
	}
	if got, want := *dst["CMP002"], (Stats{Impressions: 20, Clicks: 2, Conversions: 0, SpendCents: 100}); got != want {
		t.Errorf("inserted CMP002 = %+v, want %+v", got, want)
	}
}

func TestFinalizeMetrics(t *testing.T) {
	stats := map[string]*Stats{
		"CMP001":  {Impressions: 26000, Clicks: 640, Conversions: 27, SpendCents: 9370},
		"CMP004":  {Impressions: 1000, Clicks: 50, Conversions: 0, SpendCents: 1000}, // zero conversions
		"CMPZERO": {Impressions: 0, Clicks: 0, Conversions: 0, SpendCents: 0},        // zero impressions
	}
	got := map[string]Campaign{}
	for _, c := range Finalize(stats) {
		got[c.ID] = c
	}

	if c := got["CMP001"]; !approx(c.CTR, 640.0/26000.0) || !c.HasCPA || !approx(c.CPA, 93.70/27.0) {
		t.Errorf("CMP001: CTR=%v CPA=%v HasCPA=%v", c.CTR, c.CPA, c.HasCPA)
	}
	if c := got["CMP004"]; c.HasCPA {
		t.Errorf("CMP004 has zero conversions; HasCPA should be false")
	}
	if c := got["CMPZERO"]; c.CTR != 0 {
		t.Errorf("CMPZERO has zero impressions; CTR should be 0, got %v", c.CTR)
	}
}

func approx(a, b float64) bool {
	d := a - b
	return d < 1e-9 && d > -1e-9
}
