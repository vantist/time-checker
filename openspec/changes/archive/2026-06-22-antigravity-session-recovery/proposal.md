## Why

為了解決 Google Antigravity sessions 因歷史原因（如先前修復前建立）遺失專案路徑 `sessions.project` 與模型名稱 `sessions.model` 的問題，並修正當 active turn 因為 Stop hook 未成功執行（例如 Ctrl+C 中斷或異常結束）而殘留為懸空狀態（`response_at IS NULL`）時，由於長駐進程在背景存活，導致後續所有新 Prompts 被拒絕記錄（開頭 `activeCount > 0` 判定該 session 仍有未結束的對話）的記錄功能卡死問題。

## What Changes

- **Session 自動修補**：在 `MaybeReconcile` 時，自動篩選並修補 `project` 或 `model` 欄位為空/NULL 的歷史 sessions。
- **超時自動關閉懸空 turn**：
  - 在 `reconcileTurn` 中，若懸空 turn 的時間大於 `idle-threshold`（預設 15 分鐘），即使其進程仍存活，也強制將其關閉。
  - 在 `RecordPrompt` 中，針對 `antigravity` 工具若偵測到 active turn 且其時間已超過 `idle-threshold`，則先自動執行 `UPDATE` 關閉舊 turn，再建立新 turn，以防止記錄卡死。

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `session-management`: 自動修補遺失的 `project` 與 `model` 欄位。
- `interrupt-reconcile`: 增加懸空/掛起 turn 的 `idle-threshold` 逾期自動 reconcile/關閉機制。

## Impact

- Affected specs:
  - `openspec/specs/session-management/spec.md`
  - `openspec/specs/interrupt-reconcile/spec.md`
- Affected code:
  - New: (none)
  - Modified:
    - `internal/reconcile/reconcile.go`
    - `internal/reconcile/reconcile_test.go`
    - `internal/recorder/recorder.go`
    - `internal/recorder/recorder_test.go`
  - Removed: (none)

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-21-brainstorm-antigravity-session-recovery.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
