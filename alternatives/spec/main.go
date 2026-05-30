// Command spec is the author's base — the original "Simple Sequential Streaming"
// specification, kept as a separate binary to benchmark head-to-head against the
// solution. It uses a value-type map with read-modify-write and float64 spend
// (see ../../DESIGN.md for why those choices lost).
//
// Usage:
//
//	spec --input ad_data.csv --output results/
package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	numFields      = 6
	readBufferSize = 64 * 1024 // 64 KiB, per the original spec
	topN           = 10
)

// CampaignStats holds running totals. No pointers are stored in the map; entries
// are mutated via read-modify-write, per the original spec.
type CampaignStats struct {
	Impressions int64
	Clicks      int64
	Spend       float64
	Conversions int64
}

// FinalMetrics is a campaign with its computed metrics, ready for sorting/output.
type FinalMetrics struct {
	CampaignID  string
	Impressions int64
	Clicks      int64
	Spend       float64
	Conversions int64
	CTR         float64
	CPA         float64
	HasCPA      bool
}

var header = []string{
	"campaign_id", "total_impressions", "total_clicks",
	"total_spend", "total_conversions", "CTR", "CPA",
}

func main() {
	input := flag.String("input", "", "path to the input CSV file (required)")
	output := flag.String("output", "results", "directory to write result CSV files into")
	flag.Parse()

	if err := run(*input, *output); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(input, output string) error {
	if input == "" {
		return fmt.Errorf("--input is required")
	}

	file, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("opening input: %w", err)
	}
	defer file.Close()

	start := time.Now()
	campaignMap := make(map[string]CampaignStats, 10000)

	reader := csv.NewReader(bufio.NewReaderSize(file, readBufferSize))
	reader.FieldsPerRecord = numFields
	reader.ReuseRecord = true

	// Skip the header row.
	if _, err := reader.Read(); err != nil {
		if err == io.EOF {
			err = nil
		}
		if err != nil {
			return fmt.Errorf("reading header: %w", err)
		}
	}

	var rowsRead, rowsSkipped int
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		rowsRead++
		if err != nil {
			rowsSkipped++
			fmt.Fprintf(os.Stderr, "skipping malformed row %d: %v\n", rowsRead, err)
			continue
		}

		impressions, err1 := strconv.ParseInt(strings.TrimSpace(record[2]), 10, 64)
		clicks, err2 := strconv.ParseInt(strings.TrimSpace(record[3]), 10, 64)
		spend, err3 := strconv.ParseFloat(strings.TrimSpace(record[4]), 64)
		conversions, err4 := strconv.ParseInt(strings.TrimSpace(record[5]), 10, 64)
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			rowsSkipped++
			fmt.Fprintf(os.Stderr, "skipping unparseable row %d\n", rowsRead)
			continue
		}

		stats := campaignMap[record[0]]
		stats.Impressions += impressions
		stats.Clicks += clicks
		stats.Spend += spend
		stats.Conversions += conversions
		campaignMap[record[0]] = stats
	}

	metrics := computeMetrics(campaignMap)

	if err := os.MkdirAll(output, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	if err := writeResult(filepath.Join(output, "top10_ctr.csv"), topByCTR(metrics)); err != nil {
		return err
	}
	if err := writeResult(filepath.Join(output, "top10_cpa.csv"), topByCPA(metrics)); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr,
		"processed %d rows (%d skipped), %d campaigns, in %s\n",
		rowsRead, rowsSkipped, len(metrics), time.Since(start).Round(time.Millisecond))
	return nil
}

func computeMetrics(campaignMap map[string]CampaignStats) []FinalMetrics {
	metrics := make([]FinalMetrics, 0, len(campaignMap))
	for campaignID, s := range campaignMap {
		m := FinalMetrics{
			CampaignID:  campaignID,
			Impressions: s.Impressions,
			Clicks:      s.Clicks,
			Spend:       s.Spend,
			Conversions: s.Conversions,
		}
		if s.Impressions > 0 {
			m.CTR = float64(s.Clicks) / float64(s.Impressions)
		}
		if s.Conversions > 0 {
			m.CPA = s.Spend / float64(s.Conversions)
			m.HasCPA = true
		}
		metrics = append(metrics, m)
	}
	return metrics
}

func topByCTR(metrics []FinalMetrics) []FinalMetrics {
	ranked := slices.Clone(metrics)
	slices.SortFunc(ranked, func(a, b FinalMetrics) int {
		if a.CTR != b.CTR {
			if a.CTR > b.CTR {
				return -1
			}
			return 1
		}
		return strings.Compare(a.CampaignID, b.CampaignID)
	})
	return head(ranked)
}

func topByCPA(metrics []FinalMetrics) []FinalMetrics {
	ranked := make([]FinalMetrics, 0, len(metrics))
	for _, m := range metrics {
		if m.HasCPA {
			ranked = append(ranked, m)
		}
	}
	slices.SortFunc(ranked, func(a, b FinalMetrics) int {
		if a.CPA != b.CPA {
			if a.CPA < b.CPA {
				return -1
			}
			return 1
		}
		return strings.Compare(a.CampaignID, b.CampaignID)
	})
	return head(ranked)
}

func head(metrics []FinalMetrics) []FinalMetrics {
	if len(metrics) > topN {
		return metrics[:topN]
	}
	return metrics
}

func writeResult(path string, rows []FinalMetrics) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer file.Close()

	bw := bufio.NewWriter(file)
	cw := csv.NewWriter(bw)
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	for _, m := range rows {
		cpa := ""
		if m.HasCPA {
			cpa = strconv.FormatFloat(m.CPA, 'f', 2, 64)
		}
		record := []string{
			m.CampaignID,
			strconv.FormatInt(m.Impressions, 10),
			strconv.FormatInt(m.Clicks, 10),
			strconv.FormatFloat(m.Spend, 'f', 2, 64),
			strconv.FormatInt(m.Conversions, 10),
			strconv.FormatFloat(m.CTR, 'f', 4, 64),
			cpa,
		}
		if err := cw.Write(record); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return bw.Flush()
}
