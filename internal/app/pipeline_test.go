package app

import (
	"strings"
	"testing"

	"aggregator/internal/aggregate"
)

const header = "campaign_id,date,impressions,clicks,spend,conversions\n"

func aggString(t *testing.T, data string, workers, chunkSize int) (map[string]*aggregate.Stats, aggregate.Report) {
	t.Helper()
	stats, rep, err := runPipeline(strings.NewReader(data), workers, chunkSize)
	if err != nil {
		t.Fatalf("runPipeline: %v", err)
	}
	return stats, rep
}

// Totals must be identical regardless of where chunk boundaries fall (this is
// the carry-forward / merge correctness test).
func TestChunkBoundariesInvariant(t *testing.T) {
	data := header +
		"CMP001,2025-01-01,12000,300,45.50,12\n" +
		"CMP002,2025-01-01,8000,120,28.00,4\n" +
		"CMP001,2025-01-02,14000,340,48.20,15\n"

	for _, chunk := range []int{1, 2, 3, 7, 13, 40, 1 << 16} {
		for _, workers := range []int{1, 4} {
			stats, rep := aggString(t, data, workers, chunk)
			if rep.RowsRead != 3 || rep.RowsSkipped != 0 || len(stats) != 2 {
				t.Fatalf("chunk=%d workers=%d: read=%d skip=%d camps=%d", chunk, workers, rep.RowsRead, rep.RowsSkipped, len(stats))
			}
			c := stats["CMP001"]
			if c.Impressions != 26000 || c.Clicks != 640 || c.SpendCents != 9370 || c.Conversions != 27 {
				t.Fatalf("chunk=%d workers=%d CMP001=%+v", chunk, workers, *c)
			}
			if d := stats["CMP002"]; d.Impressions != 8000 || d.SpendCents != 2800 {
				t.Fatalf("chunk=%d workers=%d CMP002=%+v", chunk, workers, *d)
			}
		}
	}
}

func TestEdgeCases(t *testing.T) {
	cases := []struct {
		name       string
		data       string
		read, skip int
		campaigns  int
	}{
		{"empty", "", 0, 0, 0},
		{"header only", header, 0, 0, 0},
		{"header no newline", strings.TrimRight(header, "\n"), 0, 0, 0},
		{"row without trailing newline", header + "CMP001,2025-01-01,100,10,5.00,2", 1, 0, 1},
		{"malformed skipped", header + "bad,row\n" + "CMP001,2025-01-01,100,10,5.00,2\n", 2, 1, 1},
		{"blank lines ignored", header + "\n\nCMP001,2025-01-01,100,10,5.00,2\n\n", 1, 0, 1},
		{"CRLF line endings", header + "CMP001,2025-01-01,100,10,5.00,2\r\n", 1, 0, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stats, rep := aggString(t, tc.data, 4, 8) // tiny chunk to stress splitting
			if rep.RowsRead != tc.read || rep.RowsSkipped != tc.skip || len(stats) != tc.campaigns {
				t.Errorf("read=%d skip=%d camps=%d; want %d/%d/%d",
					rep.RowsRead, rep.RowsSkipped, len(stats), tc.read, tc.skip, tc.campaigns)
			}
		})
	}
}
