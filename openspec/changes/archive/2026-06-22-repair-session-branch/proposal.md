## Why

在先前的 `antigravity-session-recovery` 與 `antigravity-session-fix` 中，`time-tracker` 修復了 Antigravity 歷史 Session 的專案路徑 (`project`) 與模型名稱 (`model`)，但在自動修補邏輯 (`repairSessions`) 中漏掉了 `branch` 欄位的修復，導致部分歷史 Session 的 Git 分支資訊為空，且每次執行 reconcile 時都會因為欄位為空而重複掃描。

## What Changes

擴大 `repairSessions` 的修補範圍，包含 `branch` 欄位。
1. 當 session 的 `branch` 為空或 NULL 時，利用本地私有 `gitBranch(project)` 輔助函式來解析 Git 分支。
2. 若專案非 Git 專案或解析失敗，則寫入 `"-"` 作為佔位符以防重複修補。

成功指標 (Success Criteria)：
- 若 session 記錄中 `branch` 為空且 `project` 為有效的 Git 專案目錄，自動修補邏輯執行後，該 session 的 `branch` 欄位會被更新為該 Git 專案目前的分支名稱（例如 `test-repair-branch`）。
- 若 session 記錄中 `branch` 為空且 `project` 不是 Git 專案目錄，自動修補邏輯執行後，該 session 的 `branch` 欄位會被更新為 `"-"`。

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- session-management: 自動修補歷史 Session 欄位邏輯中新增 Git 分支修復支援

## Impact

- Affected specs:
  - `session-management`
- Affected code:
  - New: (none)
  - Modified:
    - `internal/reconcile/reconcile.go`
    - `internal/reconcile/reconcile_test.go`
  - Removed: (none)

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-21-brainstorm-repair-session-branch-info.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
