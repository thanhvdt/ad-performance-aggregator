# PROMPTS.md

The prompts that drove this build, typed to an AI assistant (Claude Code). This is the
**condensed set** — the pivotal prompts and what each one led to. The **full, unedited log** and
the referenced files (the `@CLAUDE.md` spec, `parallel.md`, `overall.md`, `overall2.md`) are in
[`raw-prompts/`](raw-prompts). Prompts are reproduced as typed; numbers match the full log.

---

**#2 — the approach**
> okay while the file is downloading lets solve the problem, im thinking read row by row, what do you suggest?

→ settled on a single streaming pass (memory O(distinct campaigns)); Go, standard library only.

**#3 — structure**
> dont mix the program with the problem, put the code inside aggregator folder, plan what we will do first

→ separated the solution from the challenge files; plan before code.

**#6 — benchmark against the author's spec** ([the spec](raw-prompts/spec.md))
> check the @CLAUDE.md and see if you can optimize your version, if different follow and simple benchmark for both versions, processing time and highest mem usage

→ built the spec's design and benchmarked it head-to-head (`baseline` vs `spec`).

**#7 — let the data decide**
> can you do some exploration with the data, i feel like the version B is too careful about the data accuracy like trimming quite space, using float kinda stuff

→ profiled all 26.8M rows; the data is clean and fixed-shape → drop defensive parsing, use integer cents.

**#8 — the synthesis**
> yup make a version C, based on all the knowledge and benchmark, and update the CLAUDE.md to include all your knowledge you can wipe the version B

→ the byte-level parser, `tuned` (~3× faster, ~5 MB).

**#9 — go parallel** ([parallel.md](raw-prompts/parallel.md))
> read the @parallel.md , and based on your knowledge, anyway to optimize more?

→ the concurrent chunked worker pool — the shipped **solution** (a further ~6× wall-clock).

**#11 — bound the memory** ([overall.md](raw-prompts/overall.md))
> read the @overall.md , and based on your knowledge, anyway to optimize more?

→ adopted a soft memory limit (measured); rejected aggressive `GOGC` and map pre-population.

**#15 — chase the last few %** ([overall2.md](raw-prompts/overall2.md))
> seems like the D also improve right, read the @overall2.md and based on you knowledge, do you think we can improve, just to explore

→ explored perfect-hash + mmap; measured and rejected (overfit / wrecks the memory metric).

**#16 — objective benchmarks**
> okay let stop now, next based on the dataset, generate different dataset, with the same variance, everything, just different in size to have objective benchmarks

→ built `tools/datagen` and ran the 128 MB → 4 GB scaling sweep.

---

A longer refactor/review phase followed (clean-code passes, package restructuring, naming, docs).
The complete prompt log is in [`raw-prompts/prompts.md`](raw-prompts/prompts.md).
