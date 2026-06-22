# Detailed Token & Cost Categories Breakdown

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

Currently, the classification details in the `serve` (web dashboard) and `report` (CLI command) are combined or missing from several aggregation tables (By Model & Role, By Project, By Agent, By Work Item, and Sessions log). Users need detailed visibility into all four token categories (Input, Output, Cache Read, Cache Create) across all reporting metrics to understand actual AI consumption patterns and optimize pricing model inputs.

## Decision

Integrate comprehensive token and cost breakdowns into all five aggregation tables in both `tt report` and the `serve` dashboard. The CLI report will display all columns directly by default and support an output file path via `--output` / `-o`. The web dashboard will display the main compact columns with interactive tooltips displaying the 4-category token breakdown on hover.

## Rationale

This approach provides full detail visibility for advanced reporting needs while resolving horizontal display limits on web dashboards using high-quality CSS tooltips. Adding a file output flag to `tt report` enables easy report sharing and backup without terminal redirect hacks.

## Approach

1. **Struct Extension**: Add `InputTokens`, `OutputTokens`, `CacheReadTokens`, and `CacheCreationTokens` fields to Go reporting structs (`ProjectSummary`, `AgentSummary`, `GroupResult`, and `SessionRow`).
2. **Aggregation Logic**: Update `report.Query` to populate these newly added fields in all group maps (`projMap`, `agentMap`, `groups`, `sessMap`).
3. **CLI Direct Display**: Update `FormatText` in `internal/report/report.go` to print all four token category columns directly for all tables by default.
4. **CLI File Output**: Add a `--output` / `-o` string flag to the `report` Cobra command to write the output directly to a file when specified.
5. **Dashboard Hover Tooltips**: Update `internal/report/html.go` with CSS tooltips and wire up JS to add detailed `data-tooltip` titles displaying all 4 categories of token breakdown on hover.

## Design Notes

### Go Reporting Struct Modifications
- Update `internal/report/report.go` to:
  - Add cache fields to `ProjectSummary`, `AgentSummary`, `GroupResult`, and `SessionRow` structs.
  - Accumulate turn-level `r.inputTok`, `r.outputTok`, `r.cacheRead`, `r.cacheCreate` into corresponding groups inside `Query()`.
  - Format columns in `FormatText` with updated layout headers and widths to align columns.

### CLI File Output Implementation
- Add standard flag `-o` / `--output` to `reportCmd` in `cmd/tt/report_cmd.go`.
- If set, perform `os.WriteFile(path, data, 0600)` with secure permissions instead of calling `fmt.Print`.

### Web Dashboard Tooltips
- Define `.tooltip` class in CSS within `dashboardHTML`.
- Update tables in HTML to include appropriate token/tooltip headers.
- Update javascript `render` function to construct detailed HTML tooltips for cells.

## Insights to Capture

- `design.md`: Supports detailed multi-category token tracking in all tables and dashboard views.
- `specs/report-query/spec.md`: Requires all reporting structures to accumulate input, output, cache read, and cache creation tokens.
- `proposal.md`: Expand token aggregation fields and support text/JSON file export.
- `tasks.md`: Implement struct field expansion, query logic accumulation, FormatText update, file output flag, and dashboard CSS/JS tooltips.

## Open Questions

None.
