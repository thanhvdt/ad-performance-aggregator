# DESIGN — Ad Performance Aggregator

Deep notes on how the solution was built and why. The short usage guide is in
[`README.md`](README.md); the prompts that drove the build are in [`PROMPTS.md`](PROMPTS.md)
(full log in [`raw-prompts/`](raw-prompts)); benchmark logs in
[`benchmarks/results.md`](benchmarks/results.md).

The submitted solution is **`cmd/aggregator`** — a parallel, chunked byte parser. It was the
end of a deliberate, benchmark-driven progression (baseline → spec → tuned → solution); the earlier versions are
kept under [`alternatives/`](alternatives) so every number below is reproducible.

---

## Repository layout

```
aggregator/                          # Go module, zero third-party dependencies
  cmd/aggregator/   # THE solution — a thin main() over internal/app
  internal/
    aggregate/      # pure domain — Stats, Add/Merge, Finalize, ranking (no I/O, no imports)
    csvio/          # the CSV format — ParseRow (read) + WriteRankings/WriteCSV (write)
    app/            # CLI wiring (Run/config) + the parallel pipeline (chunk → workers → merge)
  alternatives/     # the lineage kept for comparison: baseline, spec, tuned
  tools/datagen/    # benchmark tool — synthetic CSVs (not part of the solution; shares no code)
  results/          # output: top10_ctr.csv, top10_cpa.csv
  benchmarks/       # benchmark logs
```

Dependencies point one way: `app → {aggregate, csvio}`, `csvio → aggregate`; `aggregate`
depends on nothing. The three packages are responsibility-aligned — **domain** (aggregate),
**format** (csvio), **orchestration** (app). The solution and `tuned` build on both shared
packages and differ only in ingestion; `baseline` reuses the domain but reads via its own
`encoding/csv`; `spec` is intentionally standalone (its own types) to benchmark the naive
design honestly.

---

## The data (measured over all 26,843,544 rows)

A single `awk` profiling pass drove every design decision. Re-verify if the dataset changes —
the byte parser's safety depends on these facts:

| Property | Value |
|---|---|
| Data rows | 26,843,544 |
| Distinct campaigns | 50 (`CMP001`..`CMP050`) |
| Field count | always exactly 6 — no ragged rows, no embedded/quoted commas |
| Quote characters | 0 — there is no CSV quoting at all |
| Whitespace-padded / empty / negative / bad-date | 0 / 0 / 0 / 0 |
| Spend format | plain decimal, only 1 or 2 fractional digits (never 3+, no sci-notation) |
| `conversions == 0` | 326,361 rows (~1.2%) — the one real edge case (CPA null / excluded) |
| Max per-row impressions / spend | 50,000 / 4976.81 (tiny; per-row overflow impossible) |

**Consequences:**
- No quoting + fixed 6-field shape ⇒ a hand-rolled byte parser is provably safe; `encoding/csv`
  is pure overhead (it allocates a fresh string per row, 26.8M times).
- Spend is always ≤ 2 decimals ⇒ every value is **exact in integer cents** — no rounding loss.
- The data is pristine, so defensive input-cleaning (e.g. `TrimSpace`) is dead weight — but
  malformed-row handling (skip + count) is still implemented for robustness.

---

## Core design

- **Single streaming pass.** Read row by row; never load the file. State is a map
  `campaign_id → totals`, so **memory is O(distinct campaigns)**, independent of file size.
- **Spend as integer cents** (`int64`), not `float64`. Float addition over millions of rows
  drifts; integer cents is exact and, given ≤2-decimal inputs, lossless.
- **`map[string]*Stats`, one lookup per row** (mutate via the pointer). A value-type map with
  read-modify-write does *two* hashes per row — measurably slower.
- **Derived metrics:** `CTR = clicks/impressions` (0 when impressions == 0);
  `CPA = spend/conversions`, **null when conversions == 0** (excluded from the CPA ranking).
- **Deterministic output:** CTR/CPA ties broken by `campaign_id`.
- **Output:** CTR to 4 decimals, CPA and spend to 2 decimals; null CPA is an empty field.
  Columns: `campaign_id,total_impressions,total_clicks,total_spend,total_conversions,CTR,CPA`.
- **`ParseRow`** (in `internal/csvio`, `rowparse.go`) parses a row in one left-to-right pass over
  the bytes via small helpers (`field`/`uintField`/`centsField`/`lastUint`) — no `encoding/csv`,
  no `strconv`/`ParseFloat`. With the compiler's no-alloc `map[string(b)]` lookup the hot loop
  allocates essentially nothing (only ~50 campaign-key inserts in total).
- **The solution** wraps that parser in a worker pool: the reading goroutine pulls the file in
  512 KiB blocks (split on newlines, partial line carried forward), workers parse blocks into
  **worker-local maps** (no shared state, no locks), and a fan-in step merges them. A soft
  memory limit (`debug.SetMemoryLimit`, auto by default) keeps peak RSS low.

---

## The evolution (1 GB file, warm cache, best of 3)

The names trace the **lineage**: *baseline* was Claude's first cut, *spec* was the author's
original specification, *tuned* is the two combined with what data profiling revealed (single
thread), and the *solution* is *tuned* plus concurrency.

| Version | Where | Ingestion | Wall time | Peak RSS |
|---|---|---|---:|---:|
| Baseline | `alternatives/baseline` | `encoding/csv`, int-cents | 5.08 s | ~11 MB |
| Spec | `alternatives/spec` | `encoding/csv`, value-map, **float**, TrimSpace | 5.61 s | ~11 MB |
| Tuned | `alternatives/tuned` | manual byte parser, single thread | ~1.55 s | ~5 MB |
| **Solution** | **`cmd/aggregator`** | **parallel 512 KiB chunks + byte parser** | **~0.25 s** | ~50 MB |

All four produce **byte-identical** output. Two big levers: dropping `encoding/csv` and its
26.8M per-row string allocations (baseline→tuned, ~3×), then parallelizing the CPU-bound parse
across all cores (tuned→solution, ~6×). Tuned is the leanest (~5 MB); the solution is the
fastest. The solution's peak RSS (~50 MB, varying ~30–80 MB run-to-run because the GC barely
cycles in a sub-second burst) is bounded by a soft memory limit — without it, peak drifts to
~75–170 MB. If absolute memory frugality mattered more than wall-clock, `alternatives/tuned` at
~5 MB / 1.55 s would be the pick.

---

## Cross-size scaling (`tools/datagen`)

To benchmark scaling objectively, `tools/datagen` generates data matching the real file's
**statistical shape** at any size — verified by re-profiling (predicted clicks sd 543 ==
observed; impressions are exactly `U[1000,50000]`):

```
impressions ~ U[1000,50000];  clicks = round(impressions·ctr), ctr ~ U[0.005,0.05]
spend       = round(clicks·cpc·100) cents, cpc ~ U[0.10,2.00]   # 1 decimal when cents%10==0
conversions = round(clicks·cvr), cvr ~ U[0,0.1085]              # ~1–2% land on 0
date        = 2025-01-01 + U[0,180] days
campaign    = CMP%0Nd, distinct count = round(rows/536_871)     # cardinality scales with size
```

Sweep (best of 2, warm cache; campaigns scale with size). Throughput is flat across a 32× size
range → clean O(rows); tuned's memory stays ~5 MB at every size. Full table in
[`benchmarks/results.md`](benchmarks/results.md).

| size | campaigns | Baseline | Tuned | Solution |
|---:|---:|---|---|---|
| 128 MB | 6 | 182 MB/s | 566 MB/s | 3459 MB/s |
| 1 GB | 51 | 191 MB/s | 664 MB/s | 4249 MB/s |
| 4 GB | 206 | 189 MB/s | 649 MB/s | 4240 MB/s |

---

## Decisions & rejected ideas

The project's discipline was: form a hypothesis, build it, **measure**, keep only wins.

- **`encoding/csv` was the bottleneck**, not the map or buffer — its per-row `string()`
  allocation dominated. Dropping it (the tuned step) was the only change that moved the needle by multiples.
- **`float64` for money is a latent bug.** Values like `64.29` aren't representable in binary
  float and 26.8M additions drift; integer cents is correct by construction. (The original
  spec's "use float64 for precision" was wrong.)
- **`TrimSpace` / heavy defensive parsing:** unnecessary (0 padded fields). Row-level
  skip-and-count is kept; per-field cleaning the data never triggers is not.
- **Pointer map beats value map** (one hash/row vs two).
- **Top-N via full sort** is fine — only ~50–200 groups; a heap would be premature.
- **Parallel chunking — kept (the solution).** Built correctly: reusable read buffer, carry the trailing
  partial line forward (no `Seek`-and-re-read), worker-local maps (no mutex), fan-in merge.
- **`uint64`/FNV-1a keys + value map "to dodge the GC" — rejected.** GC scan cost matters at
  *millions* of unique keys; we have 50–200. Hashing risks silent collisions and you still need
  the id string for output. All cost, no benefit. (Row count 26.8M and group count ~50 are
  different axes; streaming handles rows, the group count is what would justify hashing.)
- **Dual fixed-size heaps — rejected.** The "O(1) memory" claim is false: you must aggregate
  every group before metrics are final, so the slice is one entry per group anyway.
- **Pre-populating worker maps with the 50 keys — rejected.** ~500 key allocations total across
  26.8M rows (unmeasurable), and it hardcodes the id format and injects phantom campaigns.
- **`mmap` — not pursued.** Would cut the solution's wall-clock further, but resident file pages
  count toward the RSS metric (could read as ~1 GB), wrecking the memory story.

### Tuning notes (measured)
- **macOS `pprof` mislabeled the read syscall as ~91%** while `sys` time was 0.08 s — a known
  profiler artifact. We trusted A/B benchmarks over the profile, which is what stopped us from
  "optimizing" the reader (a hand-rolled block reader measured ~11% *slower* than `bufio.Scanner`).
- **One-pass `ParseRow`** beat split-then-parse (29 ns/row) and an all-`IndexByte` parser
  (46 ns/row). A fully-inlined single function hit 23.7 ns/row; the current form factors it
  into small single-job helpers (`field`/`uintField`/`centsField`/`lastUint`) for readability,
  costing ~26 ns/row (~+10% on the single-threaded *tuned* version, **none** on the parallel
  solution — its parse is dwarfed by I/O and coordination). A deliberate readability-over-micro-speed trade.
- **Soft memory limit beats low `GOGC`.** `GOGC=20` reaches ~13 MB but is ~2.6× slower (GC
  thrash); a soft limit caps peak ~30–45% at full speed. The solution defaults to an auto soft limit.

---

## Edge cases & error handling

- Missing `--input`, unreadable/missing file, un-creatable output dir → clear error, exit 1.
- Empty file, header-only (with or without trailing newline) → empty output, no error.
- Last row without a trailing newline → parsed.
- Rows spanning chunk boundaries → handled by carry-forward (covered by a test that runs the
  pipeline at 1-byte chunks).
- Malformed rows (wrong field count, non-numeric) → skipped and counted, never fatal.
- `\r\n` line endings → trailing `\r` stripped.
- `int64` accumulators: safe well past 4 GB (sums stay < 10^13, limit ~9.2×10^18).

## Verify

```bash
cd aggregator
go build ./... && go vet ./... && go test ./...
go run ./cmd/aggregator --input ad_data.csv --output results
```
Expect `50 campaigns, 0 skipped`. The solution and all `alternatives/*` produce identical CSVs.
