// Command aggregator computes per-campaign advertising metrics from a large CSV
// and writes the top-10 campaigns by CTR and by lowest CPA. All logic lives in
// internal/app; this is a thin entry point.
//
// Usage:
//
//	aggregator --input ad_data.csv --output results/
package main

import (
	"fmt"
	"os"

	"aggregator/internal/app"
)

func main() {
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
