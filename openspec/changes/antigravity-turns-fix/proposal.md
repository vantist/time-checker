## Why

在 Google Antigravity 整合中，使用者回報資料庫中的 `turns` 與 `turn_model_usages` 紀錄有兩大異常：
1. **筆數異常過多**：一個 session 內只執行了幾次 user prompt，卻產生了數十個 `turns` 紀錄，且多數 `response_at` 為 `NULL`。這是因為 Antigravity 的 `PreInvocation` Hook 是在每一次 LLM 調用（即 Agent 的每一個步驟）時觸發，而非僅在 User Prompt 提交時。但 `Stop` Hook 僅在整個執行命令結束時觸發一次。這導致多次 prompt 被記錄為新 turn，但只有最後一筆 turn 能被 response 更新，其他都懸空（dangling）。
2. **數據皆為 0 與 unknown**：`turns` 與 `turn_model_usages` 的 Token 消耗皆為 0，且 Model 欄位顯示為 `"unknown"`。這是因為 Antigravity 的 `transcript.jsonl` 不包含 LLM Token 數據，其 `type` 為 `"PLANNER_RESPONSE"`，而 `internal/transcript/extract.go` 假設其與 Claude Code 格式相同（具有 `type == "assistant"` 與 `message.usage`），因而解析出 `"unknown"` 且 token 為 0。

## What Changes

1. **Turn 紀錄去重**：在 `RecordPrompt` 中，針對 `tool == "antigravity"` 的請求，若該 session 已有尚未關閉的 turn（`response_at IS NULL`），則不重複插入新的 turn，將整個 Agent 的多個思考步驟合併為單一 Turn 紀錄。
2. **Model 正確取得與路徑修正**：比照 Copilot CLI 邏輯，將無法從 log 取得的 Token 數記錄為 0。但從 `~/.gemini/antigravity-cli/settings.json` 中讀取 `"model"` 鍵值並常態化（如 `"Gemini 3.5 Flash (Medium)"` -> `"gemini-3.5-flash"`）填入 model 欄位。更新 `ResolvePath` 優先尋找 `antigravity-cli` 目錄。
3. **Reconcile 與 /clear/resume 相容支援**：修正 Reconcile 與日誌解析流程，允許對零 Token 消耗的工具（如 Antigravity）進行 Reconcile 補救與 Turn 關閉，並確保在 `/clear` 邊界或 `--resume` 會話重建時，時間與 Token 統計不發生異常。

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `event-recording`: 調整 `RecordPrompt` (記錄 prompt 事件) 規格，針對 `antigravity` 等多步驟工具，當已有未關閉的 turn（`response_at` 為 NULL）時，不重覆插入新的 turn。
- `model-cost-tracking`: 修正 Antigravity 日誌解析規格，從 `settings.json` 讀取 model，將無法從日誌中讀取的 token 記錄為 0，並修正 brain 檔案路徑解析。同時支援對零 Token 消耗的 Turn 進行 Reconcile 修補與關閉。

## Impact

- Affected specs:
  - `openspec/specs/event-recording/spec.md`
  - `openspec/specs/model-cost-tracking/spec.md`
- Affected code:
  - New: (none)
  - Modified:
    - `internal/recorder/recorder.go`
    - `internal/transcript/antigravity_transcript.go`
    - `internal/transcript/provider.go`
    - `internal/reconcile/reconcile.go`
    - `internal/reconcile/reconcile_test.go`
    - `internal/transcript/antigravity_transcript_test.go`
    - `internal/recorder/recorder_test.go`
  - Removed: (none)

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-21-brainstorm-antigravity-turns-fix.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
