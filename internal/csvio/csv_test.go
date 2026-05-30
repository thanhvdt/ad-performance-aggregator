package csvio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aggregator/internal/aggregate"
)

func TestWriteCSVFormat(t *testing.T) {
	rows := []aggregate.Campaign{
		{ID: "CMP001", Stats: aggregate.Stats{Impressions: 26000, Clicks: 640, Conversions: 27, SpendCents: 9370}, CTR: 0.024615, CPA: 3.4704, HasCPA: true},
		{ID: "CMP004", Stats: aggregate.Stats{Impressions: 1000, Clicks: 50, Conversions: 0, SpendCents: 1000}, CTR: 0.05, HasCPA: false},
	}
	var b strings.Builder
	if err := WriteCSV(&b, rows); err != nil {
		t.Fatal(err)
	}
	want := "campaign_id,total_impressions,total_clicks,total_spend,total_conversions,CTR,CPA\n" +
		"CMP001,26000,640,93.70,27,0.0246,3.47\n" + // spend 2dp, CTR 4dp, CPA 2dp
		"CMP004,1000,50,10.00,0,0.0500,\n" // null CPA -> empty field
	if got := b.String(); got != want {
		t.Errorf("WriteCSV mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestWriteRankings(t *testing.T) {
	dir := t.TempDir()
	campaigns := []aggregate.Campaign{
		{ID: "CMP001", Stats: aggregate.Stats{Impressions: 1000, Clicks: 50, Conversions: 10, SpendCents: 2000}, CTR: 0.05, CPA: 2.0, HasCPA: true},
	}
	if err := WriteRankings(dir, campaigns, 10); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"top10_ctr.csv", "top10_cpa.csv"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if !strings.HasPrefix(string(data), "campaign_id,") {
			t.Errorf("%s missing header", name)
		}
		if !strings.Contains(string(data), "CMP001") {
			t.Errorf("%s missing campaign row", name)
		}
	}
}
