## Context

Currently, the reporting functions (`tt report` and the `tt serve` web dashboard) aggregate and display token usage in a simplified format (typically only combining input/output tokens). Cache read and cache creation tokens are missing or combined. Advanced users need visibility into all four token categories (Input, Output, Cache Read, Cache Creation) to evaluate and optimize pricing model inputs.

## Goals / Non-Goals

**Goals:**
- Extend Go reporting structures to support detailed cache and token fields (`InputTokens`, `OutputTokens`, `CacheReadTokens`, `CacheCreationTokens`).
- Aggregate all four token categories across all reporting metrics (`report.Query`).
- Update `tt report` CLI outputs to display the four-category breakdown for all tables.
- Add file export support (`-o` / `--output`) in `tt report` to write output to a file directly.
- Add tooltip support in the `serve` dashboard to display the detailed four-category breakdown on hover, keeping the table layout compact.

**Non-Goals:**
- Modifying the underlying SQLite database schema (which already stores cache read/creation tokens at the turn level).
- Changing pricing calculation formulas.

## Decisions

### 1. Extends Go Structs and SQL Aggregations
We will add `InputTokens`, `OutputTokens`, `CacheReadTokens`, and `CacheCreationTokens` fields to Go structs: `ProjectSummary`, `AgentSummary`, `GroupResult`, and `SessionRow`.
We will update `report.Query` to aggregate all 4 fields during group iterations (`projMap`, `agentMap`, `dailyMap`, `groupByWorkItem`, and `sessMap`).

*Alternatives considered:*
- Storing aggregated tokens as a formatted string in all structs.
  - *Why rejected:* Format-on-read limits dashboard rendering and JSON API extensibility. Storing fields as raw integers is cleaner.

### 2. CLI FormatText Column Extension
We will extend columns in `FormatText` for the following sections:
- **By Model & Role**: Display `Input`, `Output`, `Cache Read`, `Cache Create` separately instead of just `Input Tokens` and `Output Tokens`.
- **Daily**: Display columns for `Input`, `Output`, `Cache Read`, `Cache Create` separately.
- **By Project**: Change `Tokens (I/O)` to show all four categories or separate columns.
- **By Agent**: Include all 4 categories.
- **By Work Item**: Add a `Tokens` column displaying all 4 categories.
- **Sessions**: Add a `Tokens` column displaying all 4 categories.

*Format chosen for table cells with space constraints:*
`Input / Output / Cache Read / Cache Create` (e.g. `1,234 / 567 / 100 / 50`).

### 3. CLI File Output Flags
We will add `-o` / `--output` to `cmd/tt/report_cmd.go` using Cobra flags. If provided, we write the formatted string (JSON or text) directly using `os.WriteFile(path, content, 0600)`.

### 4. Interactive Web Tooltips
Instead of overloading web table layouts with 4 separate token columns, we will introduce a CSS-based tooltip. The main column will show total tokens (Input + Output), and hovering over it will display the 4-category breakdown.

*CSS for Tooltips:*
Use `white-space: pre-line` inside absolute-positioned `::after` tooltips to support clean multi-line display of token details.

## Risks / Trade-offs

- **[Risk]** Large tables in CLI report might overflow horizontally on standard terminal widths.
  - *Mitigation:* Keep header titles short (e.g. `Input`, `Output`, `Cache Rd`, `Cache Cr`) and truncate project/agent names dynamically if needed.
