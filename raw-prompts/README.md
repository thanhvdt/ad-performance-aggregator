# raw-prompts

The complete prompt provenance for this project. The curated highlights are in
[`../PROMPTS.md`](../PROMPTS.md); this folder keeps everything.

| File | What it is |
|---|---|
| [`prompts.md`](prompts.md) | the full, unedited prompt log — every prompt in order |
| [`spec.md`](spec.md) | the original `@CLAUDE.md` referenced in prompt #6 — the author's base spec (source of `alternatives/spec`) |
| [`parallel.md`](parallel.md) | `@parallel.md` from prompt #9 — the concurrent-design + high-cardinality pitch that led to the parallel solution |
| [`overall.md`](overall.md) | `@overall.md` from prompt #11 — GC-tuning / pre-populate-maps advice (led to the soft memory limit; the rest measured and rejected) |
| [`overall2.md`](overall2.md) | `@overall2.md` from prompt #15 — perfect-hash + mmap "dark arts" (explored, rejected) |

These reference files are external advisor notes that were pasted in and evaluated during the
build; [`../DESIGN.md`](../DESIGN.md) records which of their ideas were kept versus measured and
dropped.
