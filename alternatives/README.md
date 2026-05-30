# Alternatives (comparison versions)

These are **not** the submitted solution — the solution is [`cmd/aggregator`](../cmd/aggregator)
(the parallel byte-parser). They are the earlier steps in the lineage, kept so the benchmark
story in [`../DESIGN.md`](../DESIGN.md) is reproducible:

| Dir | Lineage | Ingestion | ~1 GB time | Peak RSS |
|-----|---------|-----------|-----------:|---------:|
| [`baseline`](baseline) | Claude's base | `encoding/csv` (clean, idiomatic) | 5.1 s | ~11 MB |
| [`spec`](spec) | author's base | `encoding/csv`, value-map, float spend | 5.6 s | ~11 MB |
| [`tuned`](tuned) | baseline + spec + data findings | manual byte parser, single thread | 1.5 s | ~5 MB |

The shipped solution is `tuned` **+ concurrency**. All produce **byte-identical** output, and
share the post-processing in `internal/aggregate` (domain) + `internal/csvio` (CSV read/write;
`baseline` uses its own reader). Run any with the same flags, e.g.
`go run ./alternatives/tuned --input ad_data.csv --output results`.
