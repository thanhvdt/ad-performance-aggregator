package app

import (
	"bytes"
	"io"
	"sync"

	"aggregator/internal/aggregate"
	"aggregator/internal/csvio"
)

// workerResult is a worker's local aggregation handed back for merging.
type workerResult struct {
	stats  map[string]*aggregate.Stats
	report aggregate.Report
}

// runPipeline streams r through the parallel pipeline and returns the merged
// per-campaign totals. readChunks reads the input into blocks of whole lines;
// the worker goroutines (parseChunks) aggregate those blocks into their own
// local maps with no shared state — so no locking — and the maps are merged at
// the end.
func runPipeline(r io.Reader, workers, chunkSize int) (map[string]*aggregate.Stats, aggregate.Report, error) {
	jobs := make(chan []byte, workers*2)
	results := make(chan workerResult, workers)

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			results <- parseChunks(jobs)
		})
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	// readChunks runs here and closes jobs when the input is exhausted.
	chunkErr := readChunks(r, jobs, chunkSize)

	master := make(map[string]*aggregate.Stats)
	var rep aggregate.Report
	for res := range results {
		rep.RowsRead += res.report.RowsRead
		rep.RowsSkipped += res.report.RowsSkipped
		aggregate.Merge(master, res.stats)
	}
	if chunkErr != nil {
		return nil, rep, chunkErr
	}
	return master, rep, nil
}

// readChunks reads r in chunkSize blocks, drops the header line, and forwards
// only whole lines to jobs — carrying any trailing partial line into the next
// block so rows are never split. It closes jobs when done. Header bytes are
// discarded until the first newline, so the header may span several blocks.
func readChunks(r io.Reader, jobs chan<- []byte, chunkSize int) error {
	defer close(jobs)

	buf := make([]byte, chunkSize)
	var leftover []byte
	headerStripped := false

	for {
		n, err := io.ReadFull(r, buf)
		if n > 0 {
			chunk := buf[:n]
			if !headerStripped {
				hi := bytes.IndexByte(chunk, '\n')
				if hi < 0 {
					chunk = nil // still inside the header line; discard and read on
				} else {
					chunk = chunk[hi+1:] // keep only the bytes after the header
					headerStripped = true
				}
			}
			if nl := bytes.LastIndexByte(chunk, '\n'); nl >= 0 {
				out := make([]byte, 0, len(leftover)+nl+1)
				out = append(out, leftover...)
				out = append(out, chunk[:nl+1]...)
				leftover = append(leftover[:0], chunk[nl+1:]...)
				jobs <- out
			} else {
				leftover = append(leftover, chunk...)
			}
		}
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				if len(leftover) > 0 {
					out := make([]byte, len(leftover))
					copy(out, leftover)
					jobs <- out
				}
				return nil
			}
			return err
		}
	}
}

// parseChunks consumes byte chunks and aggregates their rows into a local map.
func parseChunks(jobs <-chan []byte) workerResult {
	stats := make(map[string]*aggregate.Stats)
	var rep aggregate.Report

	for chunk := range jobs {
		for len(chunk) > 0 {
			var line []byte
			if nl := bytes.IndexByte(chunk, '\n'); nl >= 0 {
				line, chunk = chunk[:nl], chunk[nl+1:]
			} else {
				line, chunk = chunk, nil
			}
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			if len(line) == 0 {
				continue
			}

			rep.RowsRead++
			campaignID, impressions, clicks, spendCents, conversions, ok := csvio.ParseRow(line)
			if !ok {
				rep.RowsSkipped++
				continue
			}
			s := stats[string(campaignID)]
			if s == nil {
				s = &aggregate.Stats{}
				stats[string(campaignID)] = s
			}
			s.Add(impressions, clicks, spendCents, conversions)
		}
	}
	return workerResult{stats: stats, report: rep}
}
