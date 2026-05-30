// Command datagen produces synthetic ad-performance CSVs that match the original
// dataset's statistical shape, at any target size, for objective cross-size
// benchmarking of the aggregators.
//
// The model was reverse-engineered from the real file by profiling and matches
// its means, ranges, standard deviations, spend-decimal split, zero-conversion
// fraction, and ~38.9 B average row length (see DESIGN.md):
//
//	impressions ~ U[1000, 50000]
//	clicks      = round(impressions * ctr),  ctr ~ U[0.005, 0.05]
//	spend       = round(clicks * cpc * 100) cents, cpc ~ U[0.10, 2.00]
//	conversions = round(clicks * cvr),       cvr ~ U[0, 0.1085]
//
// Distinct campaigns scale with size (the original's ~50 per GB) so the map,
// sort, and memory grow with the data instead of staying at a fixed 50.
//
// Usage:
//
//	datagen --output data.csv --size-mb 512
//	datagen --output data.csv --rows 1000000 --campaigns 20 --seed 7
package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
)

// avgRowLen is the measured average row length of the source data, used to turn
// a target size into an approximate row count (for sizing and cardinality).
const avgRowLen = 38.9

// rowsPerCampaign is the source rate (~26.84M rows / 50 campaigns), used to scale
// distinct campaigns proportionally to size.
const rowsPerCampaign = 536_871

var header = []byte("campaign_id,date,impressions,clicks,spend,conversions\n")

func main() {
	output := flag.String("output", "", "output CSV path (required)")
	sizeMB := flag.Int("size-mb", 0, "generate until the file reaches this many MB")
	rows := flag.Int64("rows", 0, "generate exactly this many data rows (overrides --size-mb)")
	campaigns := flag.Int("campaigns", 0, "distinct campaigns (0 = auto, proportional to size)")
	seed := flag.Int64("seed", 1, "PRNG seed for reproducible output")
	flag.Parse()

	if err := run(*output, *sizeMB, *rows, *campaigns, *seed); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(output string, sizeMB int, rows int64, campaigns int, seed int64) error {
	if output == "" {
		return fmt.Errorf("--output is required")
	}
	targetBytes := int64(sizeMB) << 20
	if rows <= 0 && targetBytes <= 0 {
		return fmt.Errorf("provide --rows or --size-mb")
	}

	// Approximate the eventual row count to size the campaign set.
	estRows := rows
	if estRows <= 0 {
		estRows = int64(float64(targetBytes) / avgRowLen)
	}
	if campaigns <= 0 {
		campaigns = int(math.Round(float64(estRows) / rowsPerCampaign))
		if campaigns < 1 {
			campaigns = 1
		}
	}

	ids := buildCampaignIDs(campaigns)
	dates := buildDates()

	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("creating %s: %w", output, err)
	}
	defer file.Close()

	bw := bufio.NewWriterSize(file, 1<<20)
	if _, err := bw.Write(header); err != nil {
		return err
	}

	// Stop on the row count in --rows mode, otherwise on the byte target.
	done := func(produced, written int64) bool {
		if rows > 0 {
			return produced >= rows
		}
		return written >= targetBytes
	}

	rng := rand.New(rand.NewSource(seed))
	var written, produced int64
	line := make([]byte, 0, 64)
	for !done(produced, written) {
		line = appendRow(line[:0], ids, dates, rng)
		n, err := bw.Write(line)
		if err != nil {
			return err
		}
		written += int64(n)
		produced++
	}

	if err := bw.Flush(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %d rows, %d campaigns, %.1f MB to %s\n",
		produced, len(ids), float64(written)/(1<<20), output)
	return nil
}

// appendRow draws one row from the model and appends its CSV encoding to dst.
func appendRow(dst []byte, ids [][]byte, dates [][]byte, rng *rand.Rand) []byte {
	impressions := int64(1000 + rng.Intn(49001)) // U[1000, 50000]
	ctr := 0.005 + rng.Float64()*0.045           // U[0.005, 0.05]
	clicks := int64(math.Round(float64(impressions) * ctr))
	cpc := 0.10 + rng.Float64()*1.90 // U[0.10, 2.00]
	cents := int64(math.Round(float64(clicks) * cpc * 100))
	cvr := rng.Float64() * 0.1085 // U[0, 0.1085]
	conversions := int64(math.Round(float64(clicks) * cvr))

	dst = append(dst, ids[rng.Intn(len(ids))]...)
	dst = append(dst, ',')
	dst = append(dst, dates[rng.Intn(len(dates))]...)
	dst = append(dst, ',')
	dst = strconv.AppendInt(dst, impressions, 10)
	dst = append(dst, ',')
	dst = strconv.AppendInt(dst, clicks, 10)
	dst = append(dst, ',')
	dst = appendSpend(dst, cents)
	dst = append(dst, ',')
	dst = strconv.AppendInt(dst, conversions, 10)
	dst = append(dst, '\n')
	return dst
}

// appendSpend writes cents as dollars with one decimal when the hundredths digit
// is zero (e.g. 4820 -> "48.2") and two decimals otherwise (6429 -> "64.29"),
// reproducing the source's ~90/10 two-vs-one-decimal split.
func appendSpend(dst []byte, cents int64) []byte {
	dst = strconv.AppendInt(dst, cents/100, 10)
	dst = append(dst, '.')
	rem := cents % 100
	if rem%10 == 0 {
		return append(dst, byte('0'+rem/10))
	}
	return append(dst, byte('0'+rem/10), byte('0'+rem%10))
}

// buildCampaignIDs returns n fixed-width ids (CMP001, CMP002, ...), wide enough
// that every id in this file is the same length.
func buildCampaignIDs(n int) [][]byte {
	width := len(strconv.Itoa(n))
	if width < 3 {
		width = 3
	}
	ids := make([][]byte, n)
	for i := range ids {
		ids[i] = []byte(fmt.Sprintf("CMP%0*d", width, i+1))
	}
	return ids
}

// buildDates returns every date in the source's range, 2025-01-01 .. 2025-06-30.
func buildDates() [][]byte {
	daysIn := []int{31, 28, 31, 30, 31, 30} // Jan..Jun 2025 (not a leap year)
	total := 0
	for _, d := range daysIn {
		total += d
	}
	dates := make([][]byte, 0, total)
	for m, days := range daysIn {
		for d := 1; d <= days; d++ {
			dates = append(dates, []byte(fmt.Sprintf("2025-%02d-%02d", m+1, d)))
		}
	}
	return dates
}
