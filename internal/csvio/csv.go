package csvio

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"aggregator/internal/aggregate"
)

// Header is the output column order, matching the challenge's expected format.
var Header = []string{
	"campaign_id",
	"total_impressions",
	"total_clicks",
	"total_spend",
	"total_conversions",
	"CTR",
	"CPA",
}

// WriteRankings creates dir and writes the two result files: the top-n campaigns
// by CTR (top10_ctr.csv) and the top-n by lowest CPA (top10_cpa.csv).
func WriteRankings(dir string, campaigns []aggregate.Campaign, top int) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	if err := WriteFile(filepath.Join(dir, "top10_ctr.csv"), aggregate.TopByCTR(campaigns, top)); err != nil {
		return err
	}
	return WriteFile(filepath.Join(dir, "top10_cpa.csv"), aggregate.TopByCPA(campaigns, top))
}

// WriteFile writes ranked campaigns to path through a buffered writer.
func WriteFile(path string, rows []aggregate.Campaign) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer file.Close()

	bw := bufio.NewWriter(file)
	if err := WriteCSV(bw, rows); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	if err := bw.Flush(); err != nil {
		return fmt.Errorf("flushing %s: %w", path, err)
	}
	return nil
}

// WriteCSV writes the header followed by one row per campaign. Spend is rendered
// from integer cents (exact), CTR to 4 decimals, CPA to 2 decimals; a null CPA
// is written as an empty field.
func WriteCSV(w io.Writer, rows []aggregate.Campaign) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(Header); err != nil {
		return err
	}
	for _, c := range rows {
		cpa := ""
		if c.HasCPA {
			cpa = strconv.FormatFloat(c.CPA, 'f', 2, 64)
		}
		record := []string{
			c.ID,
			strconv.FormatInt(c.Impressions, 10),
			strconv.FormatInt(c.Clicks, 10),
			formatCents(c.SpendCents),
			strconv.FormatInt(c.Conversions, 10),
			strconv.FormatFloat(c.CTR, 'f', 4, 64),
			cpa,
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// formatCents renders integer cents as a fixed two-decimal dollar string without
// going through a float.
func formatCents(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}
