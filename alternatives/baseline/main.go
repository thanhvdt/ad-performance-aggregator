// Command baseline is the start of the lineage — Claude's first clean,
// stdlib-idiomatic version built on encoding/csv. It is correct and
// memory-light, but slower than the byte-level parser (alternatives/tuned)
// because encoding/csv allocates a fresh string per row. The shipped solution
// is cmd/aggregator.
//
// Usage:
//
//	baseline --input ad_data.csv --output results/
package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"aggregator/internal/aggregate"
	"aggregator/internal/csvio"
)

const (
	numFields      = 6
	readBufferSize = 1 << 20 // 1 MiB
)

func main() {
	input := flag.String("input", "", "path to the input CSV file (required)")
	output := flag.String("output", "results", "directory to write result CSV files into")
	top := flag.Int("top", 10, "number of campaigns to include in each ranking")
	flag.Parse()

	if err := run(*input, *output, *top); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(input, output string, top int) error {
	if input == "" {
		return fmt.Errorf("--input is required")
	}

	file, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("opening input: %w", err)
	}
	defer file.Close()

	start := time.Now()
	stats, report, err := streamAggregate(file)
	if err != nil {
		return fmt.Errorf("aggregating: %w", err)
	}
	campaigns := aggregate.Finalize(stats)
	if err := csvio.WriteRankings(output, campaigns, top); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr,
		"processed %d rows (%d skipped), %d campaigns, in %s\n",
		report.RowsRead, report.RowsSkipped, len(campaigns), time.Since(start).Round(time.Millisecond))
	return nil
}

// streamAggregate streams r as CSV and folds every data row into per-campaign
// totals. The first record is treated as a header and skipped. Malformed rows
// (wrong field count or unparseable numbers) are skipped and counted rather than
// aborting the run.
func streamAggregate(r io.Reader) (map[string]*aggregate.Stats, aggregate.Report, error) {
	cr := csv.NewReader(bufio.NewReaderSize(r, readBufferSize))
	cr.FieldsPerRecord = numFields
	cr.ReuseRecord = true // reuse the record's backing array; we clone keys we keep

	stats := make(map[string]*aggregate.Stats)
	var rep aggregate.Report

	// Skip the header row. An empty input is not an error: no rows, no output.
	if _, err := cr.Read(); err != nil {
		if err == io.EOF {
			return stats, rep, nil
		}
		return nil, rep, err
	}

	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		rep.RowsRead++
		if err != nil {
			rep.RowsSkipped++
			continue
		}

		impressions, err1 := strconv.ParseInt(rec[2], 10, 64)
		clicks, err2 := strconv.ParseInt(rec[3], 10, 64)
		spendCents, err3 := parseSpendCents(rec[4])
		conversions, err4 := strconv.ParseInt(rec[5], 10, 64)
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			rep.RowsSkipped++
			continue
		}

		campaignID := rec[0]
		s := stats[campaignID]
		if s == nil {
			// rec is reused on the next Read, so clone the key we keep.
			s = &aggregate.Stats{}
			stats[strings.Clone(campaignID)] = s
		}
		s.Add(impressions, clicks, spendCents, conversions)
	}

	return stats, rep, nil
}

// parseSpendCents parses a USD spend string into integer cents, rounding to the
// nearest cent. Each value is rounded once on the way in, so the running total
// stays exact instead of drifting like repeated float addition would.
func parseSpendCents(s string) (int64, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int64(math.Round(f * 100)), nil
}
