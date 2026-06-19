## Why

在目前的設計中，當 AI 工具呼叫 subagent（如 `/spex-brainstorming`）時，系統雖能擷取其 Token 消耗，但將所有 Token（包含主 Agent 與 Subagent）混在一起累加，並以主 Agent 的 Model 單價計費，這在主客使用不同 model 時會導致費用被高估。此外，資料庫中的 `turns` 與 `sessions` 表各自僅有單一 `model` 欄位，無法細分記錄一個 turn 內各個 model 的使用明細與佔比。

## What Changes

- 新增 `turn_model_usages` 關聯表，以 `(turn_id, model, is_subagent)` 為複合主鍵，細分記錄每個 model 的 Token 與費用。
- 保留 `turns` 主表的總和快取作為 pre-aggregated 欄位以相容既有彙整查詢。
- 資料庫初始化時加入 migration backfill SQL，將歷史 `turns` 資料無痛匯入新表。
- 修改 `internal/transcript` 與 `recorder`，根據各 model 個別計算費用並寫入明細，最後同步更新至 `turns` 主表。
- CLI 報表 `tt report` 新增 `─── By Model & Role ───` 統計表。
- 網頁 Dashboard (`tt serve`) 新增 By Model & Role 的折線/比例與表格區塊。

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `event-recording`: 新增單次 turn 記錄多 model 消耗（包含 is_subagent 註記）至 `turn_model_usages` 的規格與 migration 需求。
- `subagent-token-capture`: 修改 `ExtractWindow` 的回傳結構，使其能區分主客 Agent 的 model 消耗明細。
- `report-query`: 在報表輸出中新增 By Model & Role 統計明細的規格。
- `web-dashboard`: 在網頁 dashboard 中新增 By Model & Role 統計明細與圖表的規格。

## Impact

- Affected specs: `event-recording`, `subagent-token-capture`, `report-query`, `web-dashboard`
- Affected code:
  - New: (none)
  - Modified:
    - `internal/db/schema.go`
    - `internal/transcript/transcript.go`
    - `internal/recorder/response.go`
    - `internal/report/report.go`
    - `internal/report/html.go`
  - Removed: (none)

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-20-brainstorm-subagent-model-tracking.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
