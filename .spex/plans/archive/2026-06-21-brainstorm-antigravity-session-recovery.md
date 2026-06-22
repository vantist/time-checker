# Antigravity Session Recovery & Turn Lockup Fix

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). -->

## Context

使用者目前遇到兩個已建立的 Google Antigravity sessions 因歷史原因（在先前修復前建立）而遺失欄位資訊：
1. **專案路徑缺失**：`sessions.project` 為空。
2. **模型名稱缺失**：其中一個 session 的 `sessions.model` 為空。

同時，由於 Antigravity 的長駐執行程序（`agy` 進程）在背景一直存活，一旦有任何 turn 因為 `Stop` hook 未成功執行（例如 Ctrl+C 中斷或異常結束）而殘留為 `response_at IS NULL`，這筆 turn 便永遠不會被 `reconcile` 自動關閉。這導致 `RecordPrompt` 以 `activeCount > 0` 判定該 session 仍有未結束的對話，從而**拒絕為後續所有新 Prompts 建立新 turns**，造成記錄功能卡死。

## Decision

採用 **Option A：靜默自動修補與超時自動關閉** 解決方案。

### 1. 歷史 Session 背景自動修補
在 `MaybeReconcile`（`tt report` 或 `tt serve` 啟動時）執行 `repairSessions(db)` 邏輯：
* 篩選出 `project` 或 `model` 欄位為空/NULL 的 sessions。
* 讀取其 turns 下存在的 `transcript_full.jsonl`，掃描 JSON 結構中的絕對路徑（屬 Home 目錄但排除 `.gemini`、`.claude`、`.copilot`、`Library` 等系統或工具自用路徑）。
* 向上層目錄查找 `.git` 或 `go.mod` 來重建專案目錄路徑；若無則 fallback 至 `os.Getwd()`。
* 從日誌解析或 settings.json 中回填 `sessions.model`（Antigravity 預設為 `gemini-3.5-flash`）。

### 2. 進程存活超時自動關閉
* 在 `reconcileTurn` 中，若 dangling turn 已大於 `idle-threshold`（預設 15 分鐘），則即使 `process.IsAlive` 為 true 也不再跳過，強制將其關閉。
* 在 `RecordPrompt` 中，若 active turn 大於 `idle-threshold`，則先執行 SQL `UPDATE` 關閉舊 turn，再建立新 turn。

---

## Detailed Design & Implementation

### 1. `internal/reconcile/reconcile.go`
* 新增 `repairSessions(db *sql.DB) error`。
* 遞迴掃描 JSON map 尋找符合 `/Users/<name>/...` 且過濾掉排除名單的路徑字串：
  ```go
  func scanForWorkspacePath(val any) string {
      // 搜尋 "cwd", "DirectoryPath", "AbsolutePath", "path" 鍵值
      // 排除以 "." 開頭、"Library"、"Downloads"、"Desktop"、"Applications" 等路徑
  }
  ```
* 向上尋找專案根目錄：
  ```go
  func findProjectRoot(p string) string {
      // 遞迴向上層目錄找 .git 或 go.mod，查到則回傳該路徑
  }
  ```
* 修改 `reconcile(conn)` 以在最開始呼叫 `_ = repairSessions(conn)`。
* 修改 `reconcileTurn` 的進程存活判斷，加上 `idle-threshold` 超時檢查。

### 2. `internal/recorder/recorder.go`
* 在 `RecordPrompt` 中，針對 `tool == "antigravity"` 且 `activeCount > 0` 的情況，取得該 active turn 的 `prompt_at` 時間。
* 若 `time.Since(promptAt) > time.Duration(idleMin)*time.Minute`，則執行 `UPDATE turns SET response_at=? WHERE id=?` 強制關閉，隨後繼續插入新 turn。
