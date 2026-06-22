## Why

在 Google Antigravity 整合中，目前存在以下兩大問題：
1. **專案路徑未抓到**：在建立 session 時，`sessions.project` 欄位為空 `""`。這是因為 Antigravity 的 `PreInvocation` hook 的 stdin JSON payload 並不包含 `cwd` 欄位，導致 `resolvePromptInput` 解析出的 `project` 為空。
2. **其中一個 session 未抓到 model**：資料庫中有一筆 session 的 `sessions.model` 欄位為空。這是因為該 session 對應的 `agy` 處理程序目前仍在背景運行中，並未觸發 `Stop` hook（未呼叫 `RecordResponse`），而後續背景 `reconcile` 雖然修補了 turns 且填入了 `turns.model`，卻沒有同步回填 `sessions.model` 欄位。

## What Changes

1. **專案路徑 Fallback 至 `os.Getwd()`**：在 `resolvePromptInput` 中，若從 CLI 參數與 stdin JSON 均未取得 `project` 路徑，則 fallback 使用當前行程的 working directory `os.Getwd()`。
2. **建立 Prompt 時主動載入設定檔之 Model**：將 `internal/transcript/antigravity_transcript.go` 中的 `getAntigravityModel` 導出為 `GetAntigravityModel`。在 `resolvePromptInput` 中，若 `tool == "antigravity"` 且 `model == ""`，則呼叫該函數從 `settings.json` 中讀取預設模型寫入 session，避免初始 model 為空。
3. **Reconcile 成功時回填 `sessions.model`**：在 `internal/reconcile/reconcile.go` 中的 `reconcileTurn` 函數裡，若解析出的 turn model 不為空，則同步更新 `sessions.model`（僅在原本為空或為 NULL 時更新）。

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `hook-integration`: 補充說明 project path 與 model name 的解析規範。

## Impact

- Affected specs:
  - `openspec/specs/hook-integration/spec.md`
- Affected code:
  - New: (none)
  - Modified:
    - `cmd/tt/record.go`
    - `internal/transcript/antigravity_transcript.go`
    - `internal/transcript/antigravity_transcript_test.go`
    - `internal/reconcile/reconcile.go`
  - Removed: (none)

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-21-brainstorm-antigravity-session-fix.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
