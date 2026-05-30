// Package app wires the aggregator command together: it parses flags, reads and
// aggregates the input CSV through the parallel pipeline (pipeline.go), derives
// the metrics, and writes the ranked result files. The cmd/aggregator main
// function is a thin shell over app.Run.
package app

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"aggregator/internal/aggregate"
	"aggregator/internal/csvio"
)

const (
	// defaultChunkKiB is how much the chunker reads per block. A sweep showed
	// ~512 KiB is the sweet spot: near-peak throughput with low in-flight memory.
	defaultChunkKiB = 512
	// minMemLimit floors the auto memory cap so a small worker count can't set a
	// limit so tight it makes the GC thrash.
	minMemLimit = 32 << 20
)

type config struct {
	input, output       string
	top, workers        int
	chunkSize, memLimit int // bytes
}

// Run parses command-line args and runs the aggregator.
func Run(args []string) error {
	fs := flag.NewFlagSet("aggregator", flag.ExitOnError)
	input := fs.String("input", "", "path to the input CSV file (required)")
	output := fs.String("output", "results", "directory to write result CSV files into")
	top := fs.Int("top", 10, "number of campaigns to include in each ranking")
	workers := fs.Int("workers", runtime.NumCPU(), "number of parser workers")
	chunkKiB := fs.Int("chunk-kib", defaultChunkKiB, "read-chunk size in KiB")
	memLimitMiB := fs.Int("mem-limit-mib", -1, "soft memory cap in MiB (-1 = auto, 0 = off)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return execute(buildConfig(*input, *output, *top, *workers, *chunkKiB, *memLimitMiB))
}

// buildConfig normalizes flag values: clamps workers/chunk size and resolves the
// memory cap (-1 → auto, scaled to the in-flight working set; 0 → off).
func buildConfig(input, output string, top, workers, chunkKiB, memLimitMiB int) config {
	if workers < 1 {
		workers = 1
	}
	if chunkKiB < 1 {
		chunkKiB = defaultChunkKiB
	}
	chunkSize := chunkKiB << 10

	memLimit := 0
	switch {
	case memLimitMiB > 0:
		memLimit = memLimitMiB << 20
	case memLimitMiB < 0: // auto: cover the chunks in flight, with headroom
		memLimit = max(minMemLimit, 6*workers*chunkSize)
	}
	return config{input, output, top, workers, chunkSize, memLimit}
}

// execute reads & aggregates the CSV, derives metrics, then writes the result files.
func execute(cfg config) error {
	if cfg.input == "" {
		return fmt.Errorf("--input is required")
	}
	if cfg.memLimit > 0 {
		debug.SetMemoryLimit(int64(cfg.memLimit))
	}

	file, err := os.Open(cfg.input)
	if err != nil {
		return fmt.Errorf("opening input: %w", err)
	}
	defer file.Close()

	start := time.Now()
	stats, report, err := runPipeline(file, cfg.workers, cfg.chunkSize)
	if err != nil {
		return fmt.Errorf("aggregating: %w", err)
	}
	campaigns := aggregate.Finalize(stats)

	if err := csvio.WriteRankings(cfg.output, campaigns, cfg.top); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "processed %d rows (%d skipped), %d campaigns, %d workers, in %s\n",
		report.RowsRead, report.RowsSkipped, len(campaigns), cfg.workers, time.Since(start).Round(time.Millisecond))
	return nil
}
