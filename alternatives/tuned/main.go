// Command tuned combines the lineage — Claude's baseline, the author's spec, and
// what data profiling revealed — into a single-threaded byte parser. Knowing the
// data is exactly 6 unquoted fields with at-most-two-decimal spend, it parses
// rows at the byte level (csvio.ParseRow), avoiding encoding/csv's per-row string
// allocation entirely. The shipped solution (cmd/aggregator) adds concurrency on
// top of this.
//
// Usage:
//
//	tuned --input ad_data.csv --output results/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"aggregator/internal/aggregate"
	"aggregator/internal/csvio"
)

const scanBuffer = 1 << 20 // 1 MiB read/token buffer; rows are tiny, this just batches reads

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
	stats, report, err := byteAggregate(file)
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

// byteAggregate streams r line by line via bufio.Scanner and parses each row in
// one pass with csvio.ParseRow. No encoding/csv, no per-row string
// allocation: the only heap work in the hot loop is inserting a brand-new
// campaign key (50×). (A hand-rolled block reader was tried and measured slower —
// Scanner is already well-tuned and read() was never the bottleneck.)
func byteAggregate(r io.Reader) (map[string]*aggregate.Stats, aggregate.Report, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, scanBuffer), scanBuffer)

	stats := make(map[string]*aggregate.Stats)
	var rep aggregate.Report

	// Skip the header. A missing header (empty input) is not an error.
	if !sc.Scan() {
		return stats, rep, sc.Err()
	}

	for sc.Scan() {
		rep.RowsRead++
		campaignID, impressions, clicks, spendCents, conversions, ok := csvio.ParseRow(sc.Bytes())
		if !ok {
			rep.RowsSkipped++
			continue
		}

		s := stats[string(campaignID)] // compiler avoids allocating a string for the lookup
		if s == nil {
			s = &aggregate.Stats{}
			stats[string(campaignID)] = s // allocates the key once per new campaign
		}
		s.Add(impressions, clicks, spendCents, conversions)
	}

	if err := sc.Err(); err != nil {
		return nil, rep, err
	}
	return stats, rep, nil
}
