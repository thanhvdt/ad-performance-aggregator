// Package aggregate is the pure analytics domain: per-campaign running totals
// (Stats, with Add/Merge), the derived metrics (Finalize → Campaign with CTR and
// CPA), and the top-N rankings (rank.go). It has no I/O and no dependencies —
// reading/parsing rows lives in internal/csvio, orchestration in internal/app.
package aggregate

// Stats holds the running totals for one campaign. Spend is kept as integer
// cents so the sum never accumulates floating-point drift over millions of rows.
type Stats struct {
	Impressions int64
	Clicks      int64
	Conversions int64
	SpendCents  int64
}

// Add folds one row's parsed values into the totals.
func (s *Stats) Add(impressions, clicks, spendCents, conversions int64) {
	s.Impressions += impressions
	s.Clicks += clicks
	s.SpendCents += spendCents
	s.Conversions += conversions
}

// Merge adds every campaign's totals from src into dst — used to fan-in
// worker-local maps after a parallel pass.
func Merge(dst, src map[string]*Stats) {
	for campaignID, s := range src {
		if d := dst[campaignID]; d != nil {
			d.Add(s.Impressions, s.Clicks, s.SpendCents, s.Conversions)
		} else {
			dst[campaignID] = s
		}
	}
}

// Report summarizes what an aggregation pass saw.
type Report struct {
	RowsRead    int // data rows encountered (excludes the header)
	RowsSkipped int // malformed rows that were skipped
}

// Campaign is a finalized per-campaign result with its derived metrics.
type Campaign struct {
	ID string
	Stats
	CTR    float64 // clicks / impressions (0 when impressions == 0)
	CPA    float64 // spend / conversions (valid only when HasCPA is true)
	HasCPA bool    // false when conversions == 0
}

// Finalize turns the raw per-campaign totals into Campaign records with derived
// CTR and CPA. CTR is 0 when there are no impressions; CPA is marked absent
// (HasCPA == false) when there are no conversions.
func Finalize(stats map[string]*Stats) []Campaign {
	out := make([]Campaign, 0, len(stats))
	for campaignID, s := range stats {
		c := Campaign{ID: campaignID, Stats: *s}
		if s.Impressions > 0 {
			c.CTR = float64(s.Clicks) / float64(s.Impressions)
		}
		if s.Conversions > 0 {
			c.CPA = float64(s.SpendCents) / 100.0 / float64(s.Conversions)
			c.HasCPA = true
		}
		out = append(out, c)
	}
	return out
}
