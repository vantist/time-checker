# Repair Session Branch

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

在先前的 `antigravity-session-recovery` 與 `antigravity-session-fix` 中，`time-tracker` 修復了 Antigravity 歷史 Session 的專案路徑 (`project`) 與模型名稱 (`model`)，但在自動修補邏輯 (`repairSessions`) 中漏掉了 `branch` 欄位的修復，導致部分歷史 Session 的 Git 分支資訊為空，且每次執行 reconcile 時都會因為欄位為空而重複掃描。

## Decision

擴大 `repairSessions` 的修補範圍，包含 `branch` 欄位。當 session 的 `branch` 為空時，利用本地私有 `gitBranch(project)` 輔助函式來解析 Git 分支；若專案非 Git 專案，則寫入 `"-"` 作為佔位符以防重複修補。

## Rationale

- **避免重複查詢**：非 Git 專案寫入 `"-"` 可防止 `reconcile` 流程在下次執行時，因為 `branch` 為空而再度查詢該 session。
- **低耦合**：在 `internal/reconcile/reconcile.go` 中實作私有 `gitBranch`，以避免與 `recorder` 模組產生雙向依賴。

## Approach

- 在 `internal/reconcile/reconcile.go` 實作本地私有 `gitBranch` 函式。
- 將 `repairSessions` 的 SQL 查詢條件加入 `branch IS NULL OR branch = ''`。
- 修改 `repairSessions` 以在 `branch` 為空且 `project` 不為空時自動解析分支，解析失敗則寫入 `"-"`。
- 新增 `UPDATE` 語句更新 `branch` 欄位。
- 在 `internal/reconcile/reconcile_test.go` 中新增測試：
  - 初始化一個真實 Git repo，驗證 `branch` 欄位能成功修復為 `test-repair-branch`。
  - 對非 Git 專案驗證 `branch` 欄位被填入 `"-"`。

## Design Notes

### Database Update
```sql
UPDATE sessions
SET project = ?, model = ?, branch = ?
WHERE id = ?
```

### Privately Helper Function
```go
func gitBranch(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
```

## Insights to Capture

- `design.md`: 模組間功能獨立，`reconcile` package 自行實作 `gitBranch` 避免依賴 `recorder`
- `tasks.md`: 擴展 `repairSessions` query / struct 以及單元測試以包含 branch 修復

## Open Questions

None
