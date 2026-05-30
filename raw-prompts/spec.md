<!--
Provenance: this is the original CLAUDE.md that prompt #6 referenced ("check the @CLAUDE.md …").
It is the author's base specification — the source of the `spec` version (alternatives/spec).
The repo's CLAUDE.md has since been rewritten into agent notes, so the original spec is
preserved here verbatim as a prompt reference. (See ../DESIGN.md for why several of its choices
— float spend, value-map, TrimSpace, fixed-capacity map — were measured and dropped.)
-->

To give your coding agent the exact blueprint it needs to write flawless, production-ready Go code, the prompt must be highly explicit about data structures, memory behavior, error boundaries, and specific standard library configurations.

Here is the comprehensive technical specification for the **Simple (Sequential Streaming) Approach**. You can copy and paste this directly into your coding agent.

---

# Specification: High-Performance Sequential CSV Aggregator in Go

## Objective

Write a CLI application in Go that parses a large (~1GB) advertising performance CSV file sequentially, aggregates metrics per `campaign_id` using zero-allocation primitives where possible, and outputs two separate top-10 result files. The design must minimize memory footprint by streaming row-by-row and avoiding redundant string or pointer allocations.

---

## 1. Core Architecture & Data Structures

### Data Models

Define a single compact structure to hold the running aggregations. Do not use pointers for internal tracking inside the map to prevent heap fragmentation.

```go
type CampaignStats struct {
    Impressions int64
    Clicks      int64
    Spend       float64
    Conversions int64
}

```

### State Storage

Use a native Go map mapped directly to the struct value:

```go
// Initialize with a safe baseline capacity to reduce resizing overhead
campaignMap := make(map[string]CampaignStats, 10000)

```

---

## 2. Execution Workflow

### Step 1: CLI Input Validation

- Accept two flags: `--input` (path to the input CSV) and `--output` (path to the target directory).
- Validate that the input file exists.
- Ensure the target output directory exists; if not, create it using `os.MkdirAll`.

### Step 2: High-Efficiency File I/O Initialization

- Open the input file via `os.Open`. Ensure it is deferred to close properly.
- Wrap the file handle in a buffered reader to eliminate excessive syscalls:

```go
bufferedReader := bufio.NewReaderSize(file, 64 * 1024) // 64KB Buffer

```

- Initialize the CSV reader using `encoding/csv`:

```go
reader := csv.NewReader(bufferedReader)
reader.FieldsPerRecord = 6 // Strict validation of column count
reader.ReuseRecord = true   // CRITICAL: Reuses the slice memory across reads

```

### Step 3: Stream, Parse, and Mutate

1. Read and skip the header row.
2. Enter a `for` loop executing `record, err := reader.Read()`.

- Handle `io.EOF` gracefully by breaking the loop.
- If any other parsing error occurs (e.g., malformed row), log it to `os.Stderr` and continue to the next row (do not panic).

3. Extract and parse data by strict index placement:

- `record[0]` $\rightarrow$ `campaign_id` (string)
- `record[1]` $\rightarrow$ _Skip_ (Date column)
- `record[2]` $\rightarrow$ `impressions` (Parse using `strconv.ParseInt(..., 10, 64)`)
- `record[3]` $\rightarrow$ `clicks` (Parse using `strconv.ParseInt(..., 10, 64)`)
- `record[4]` $\rightarrow$ `spend` (Parse using `strconv.ParseFloat(..., 64)`)
- `record[5]` $\rightarrow$ `conversions` (Parse using `strconv.ParseInt(..., 10, 64)`)

4. **Map Mutation Logic (Zero-Allocation Insertion):**
   To update a map value without allocating new memory blocks or copying data via pointers, retrieve the entry, mutate it in-place, and re-assign it:

```go
stats := campaignMap[record[0]]
stats.Impressions += parsedImpressions
stats.Clicks += parsedClicks
stats.Spend += parsedSpend
stats.Conversions += parsedConversions
campaignMap[record[0]] = stats

```

---

## 3. Post-Processing & Metrics Computation

Once streaming completes, create an intermediate structure or map representation to calculate the computed metrics cleanly before sorting:

```go
type FinalMetrics struct {
    CampaignID  string
    Impressions int64
    Clicks      int64
    Spend       float64
    Conversions int64
    CTR         float64
    CPA         float64
    HasCPA      bool // To track if conversions > 0
}

```

### Metrics Formulae Rules

Iterate through `campaignMap`, populate a slice of `FinalMetrics`, and compute:

- $\text{CTR} = \frac{\text{Clicks}}{\text{Impressions}}$ (If `Impressions` is 0, $\text{CTR} = 0.0$).
- $\text{CPA} = \frac{\text{Spend}}{\text{Conversions}}$ (If `Conversions` is 0, set `HasCPA = false` or keep it null).

---

## 4. Sorting and Output Generation

Since the unique campaign count is bound to a reasonable size per standard marketing datasets, transform the map into a slice of `FinalMetrics` and execute the following targeted sorts using the standard `slices` package (Go 1.21+):

### File A: Top 10 Highest CTR (`top10_ctr.csv`)

- Sort descending by `CTR`.
- If two campaigns have identical CTRs, break the tie by sorting alphabetically by `CampaignID`.
- Write the top 10 elements to the output file using `encoding/csv`.
- Format float outputs to 4 decimal places for `CTR` and 2 decimal places for `CPA`.

### File B: Top 10 Lowest CPA (`top10_cpa.csv`)

- Filter out any campaign where `HasCPA` is false (conversions equal 0).
- Sort ascending by `CPA`.
- If two campaigns have identical CPAs, break the tie by sorting alphabetically by `CampaignID`.
- Write the top 10 elements to the output file.

---

## 5. Defensive Edge Cases & Robustness Rules

Instruct your agent to implement the following defensive programming layers:

- **Divide by Zero Protection:** Explicitly wrap CTR and CPA calculations in checks to prevent `NaN` or `Inf` outputs.
- **Data Cleansing:** If a numeric string contains whitespace, call `strings.TrimSpace()` before passing it to `strconv`. If parsing fails, skip the individual row rather than crashing the execution context.
- **Precision Handling:** Ensure all `Spend` and `CPA` calculations utilize standard `float64` floating-point math to adhere to the requirements. Space out output layouts cleanly using a standard comma delimiter.
