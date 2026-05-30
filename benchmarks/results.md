# Benchmark logs

**Machine:** Apple Silicon (10 cores), macOS, Go 1.26, SSD. **Method:** OS page cache warmed
(`cat file > /dev/null`), best of 3, wall time = the program's internal timer, peak RSS =
`/usr/bin/time -l` "maximum resident set size". Input: the real `ad_data.csv` (995 MB,
26,843,544 rows, 50 campaigns) unless noted. All versions produce **byte-identical** output.

## Solution (`cmd/aggregator`) on the 1 GB file

| Metric | Value |
|---|---|
| Processing time | **~0.25 s** (5 runs: 244–256 ms; 26.8 M rows → ~4.0 GB/s) |
| Peak memory | **~50 MB** (5 runs: 27–83 MB — GC-timing variance; auto soft cap, `--mem-limit-mib 0` to disable) |
| Rows skipped | 0 |
| Campaigns | 50 |

The work is CPU-bound parsing spread across all cores; total CPU time (~1.6 s) is similar to
the single-threaded parser, but wall-clock is ~6.5× lower.

## Version comparison (1 GB, same input)

| Version | Ingestion | Wall time | Peak RSS |
|---|---|---:|---:|
| Baseline `alternatives/baseline` | `encoding/csv` | 5.07 s | ~11 MB |
| Spec `alternatives/spec` | `encoding/csv`, value-map, float | 5.62 s | ~11 MB |
| Tuned `alternatives/tuned` | manual byte parser, single thread | ~1.55 s | ~5 MB |
| **Solution `cmd/aggregator`** | **parallel chunks + byte parser** | **~0.25 s** | ~50 MB |

Takeaways: dropping `encoding/csv` (its 26.8 M per-row string allocations) is the ~3× win
(baseline→tuned); parallel chunking is a further ~6× wall-clock win (tuned→solution) by using
all cores. See [`../DESIGN.md`](../DESIGN.md) for the full story and the rejected ideas.

## Scaling sweep (synthetic data from `tools/datagen`, campaigns scale with size)

Best of 2, warm cache. Confirms linear-in-rows scaling and stable throughput.

| size | campaigns | Baseline | Spec | Tuned | Solution |
|---:|---:|---|---|---|---|
| 128 MB | 6   | 0.70 s / 182 MB/s | 0.74 s / 173 | 0.23 s / 566 | 0.04 s / 3459 |
| 256 MB | 13  | 1.34 s / 192 | 1.48 s / 173 | 0.38 s / 668 | 0.06 s / 4571 |
| 512 MB | 26  | 2.68 s / 191 | 2.96 s / 173 | 0.78 s / 659 | 0.14 s / 3737 |
| 1 GB   | 51  | 5.35 s / 191 | 5.92 s / 173 | 1.54 s / 664 | 0.24 s / 4249 |
| 2 GB   | 103 | 10.83 s / 189 | 11.89 s / 172 | 3.18 s / 644 | 0.47 s / 4395 |
| 4 GB   | 206 | 21.64 s / 189 | 23.84 s / 172 | 6.31 s / 649 | 0.97 s / 4240 |

- Throughput is flat across a 32× size range (A ~190, B ~172, C ~650, D ~4000–4500 MB/s) →
  clean O(rows).
- C's memory stays ~5 MB and A/B ~11 MB at every size, independent of size *and* cardinality
  (6 → 206 campaigns) — the map/sort never bottlenecks at these cardinalities.

## Reproduce

```bash
# generate a sized dataset (matches the real file's distributions)
go run ./tools/datagen --output /tmp/bench.csv --size-mb 1024 --seed 1
# warm cache, then time + measure memory
cat /tmp/bench.csv > /dev/null
/usr/bin/time -l go run ./cmd/aggregator --input /tmp/bench.csv --output /tmp/out
```
