# Ad Performance Aggregator

A CLI that streams a large (~1 GB) advertising-performance CSV, aggregates metrics per
`campaign_id`, and writes the **top 10 campaigns by CTR** and the **top 10 by lowest CPA**.

| | |
|---|---|
| **Language** | Go 1.26 — standard library only, **zero third-party dependencies** |
| **1 GB / 26.8M rows** | **~0.25 s**, peak memory **~50 MB** |
| **Output** | `results/top10_ctr.csv`, `results/top10_cpa.csv` |

Measured with `/usr/bin/time -l`, warm cache, Apple Silicon (10 cores). Peak RSS varies
~30–80 MB run-to-run — the GC barely cycles in a sub-second burst; it's soft-capped (see Flags).

## Setup

Go 1.26+, nothing to fetch.

```bash
cd aggregator
go build ./...
```

## Run

```bash
go run ./cmd/aggregator --input ad_data.csv --output results/
# or build the binary first:
go build -o aggregator ./cmd/aggregator
./aggregator --input ad_data.csv --output results/
```

**Flags:** `--input` (required) · `--output` (default `results`) · `--top` (default 10) ·
`--workers` (default = CPU count) · `--chunk-kib` (default 512) · `--mem-limit-mib`
(`-1` auto soft cap, `0` off).

**Output** — `campaign_id,total_impressions,total_clicks,total_spend,total_conversions,CTR,CPA`:

```
CMP005,13648608306,375627610,394780333.96,20403485,0.0275,19.35
```

CTR has 4 decimals; CPA and spend 2. CPA is blank for campaigns with zero conversions, which are
also excluded from the CPA ranking.

## Docker

```bash
docker build -t ad-aggregator .
docker run --rm -v "$PWD":/data ad-aggregator --input /data/ad_data.csv --output /data/results
```

## Tests

```bash
go test ./...
```

Cover the byte parser, the metric/ranking/CSV logic, and the concurrent pipeline across chunk
boundaries and edge cases (empty / header-only / no-trailing-newline / malformed / CRLF).

## How it works

- **O(distinct campaigns) memory** — a single streaming pass folds rows into a
  `campaign_id → totals` map; nothing scales with file size.
- **Parallel** — the file is read in 512 KiB blocks (split on newlines so rows are never cut)
  and parsed across a worker pool into per-worker maps with no shared state (no locks), then merged.
- **Byte-level parsing** — the data is unquoted and fixed-shape, so rows are parsed directly from
  bytes, avoiding `encoding/csv`'s ~26.8M per-row string allocations (the dominant cost otherwise).
- **Exact money** — spend is summed in integer cents, so totals never drift like `float64` would.
- **Robust** — malformed rows are skipped and counted (not fatal); missing/empty/header-only
  files and `\r\n` endings are handled.

## More

- **[DESIGN.md](DESIGN.md)** — full design, the baseline → spec → tuned → solution evolution,
  decisions, and the ideas measured and rejected.
- **[benchmarks/results.md](benchmarks/results.md)** — benchmark logs and the 128 MB → 4 GB scaling sweep.
- **[PROMPTS.md](PROMPTS.md)** — the pivotal AI-assistant prompts that drove the build (full
  log and the referenced advisor files in **[raw-prompts/](raw-prompts)**).
- **[alternatives/](alternatives)** — the earlier versions kept for comparison: `baseline`
  (Claude's base) → `spec` (author's base) → `tuned` (the two combined with data profiling).
  The shipped solution is `tuned` **+ concurrency**.
