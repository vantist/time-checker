## Context

在先前針對 Antigravity Session 修復的實作中，`time-tracker` 透過 `internal/reconcile/reconcile.go` 中的 `repairSessions` 自動修補歷史 Session 的專案路徑 (`project`) 與模型名稱 (`model`)。然而，該自動修補邏輯中並未修復 `branch` 欄位。這導致：
1. 部分歷史 Session 的 Git 分支資訊為空。
2. 每次執行 `reconcile` 時，由於 `branch` 為空，修補邏輯會重複對這些 Session 進行資料解析與掃描。

## Goals / Non-Goals

**Goals:**
- 擴大自動修補邏輯 `repairSessions` 的範圍，使其支援對歷史 Session 的 `branch` (Git 分支) 欄位進行修復。
- 若專案為 Git 專案，應將 `branch` 修復為對應的 Git 分支名稱。
- 若專案非 Git 專案，應填入 `"-"` 作為佔位符以防重複修補與查詢。

**Non-Goals:**
- 不在此次變更中修改 `sessions` 的資料表 Schema。
- 不為非 Git 專案開發複雜的分支識別邏輯。

## Decisions

### 1. 於 `internal/reconcile` 封裝私有 Git 分支解析輔助函式

- **決策**：在 `internal/reconcile/reconcile.go` 中實作一個本地私有的 `gitBranch(dir string) string` 函式。
- **考量**：雖然 `internal/recorder` 模組可能有類似的分支讀取邏輯，但為了避免 `reconcile` 與 `recorder` 之間產生雙向/循環依賴，保持模組低耦合，決定在 `internal/reconcile` 中獨立實作該私有輔助函式。
- **實作細節**：使用 Go 標準庫 `os/exec` 執行 git rev-parse 指令（帶有 -C 參數指向專案目錄，以及 rev-parse --abbrev-ref HEAD 參數）。若執行失敗或回傳空字串，則回傳空字串 `""`。

### 2. 更新 SQL 查詢與資料修補邏輯

- **決策**：
  1. 將 `repairSessions` 中的 `SELECT` 查詢條件加入 `branch IS NULL OR branch = ''`。
  2. 在掃描 Session 列表時，若發現 `branch` 為空，且 `project` 不為空（若 `project` 原本為空，則在此處應先經過專案路徑修復邏輯），呼叫 `gitBranch(project)` 解析分支。
  3. 解析成功則使用解析到的分支；解析失敗（例如非 Git 專案）則使用 `"-"` 作為分支名稱。
  4. 使用 `UPDATE sessions SET project = ?, model = ?, branch = ? WHERE id = ?` 語句一次更新三個修補欄位。

## Risks / Trade-offs

- **[Risk] 非 Git 專案的分支查詢效率低下**
  - *Mitigation*: 藉由將無法解析分支的專案之 `branch` 欄位寫入 `"-"` 佔位符，確保下次 `reconcile` 執行時，該 Session 的 `branch` 不再是空值，從而避免再次對其執行 `gitBranch` 指令。
