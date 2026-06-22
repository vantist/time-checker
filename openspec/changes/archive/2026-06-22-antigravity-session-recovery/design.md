## Context

在先前版本的 time-tracker 中，部分 Google Antigravity sessions 建立時並未正確紀錄 `project`（專案路徑）與 `model`（模型名稱）。這導致歷史資料中存在缺失欄位的 session。

此外，Antigravity 的背景進程 `agy` 會持續存活。若發生 Ctrl+C 中斷或異常結束，導致 `Stop` hook 未能執行，DB 中就會殘留 `response_at IS NULL` 的 turn（懸空 turn）。由於 `agy` 進程仍在背景存活，`reconcile` 流程會因為進程存活而跳過對該 turn 的自動關閉，導致 `RecordPrompt` 始終判定 `activeCount > 0`，進而拒絕為該 session 建立任何後續的新 turns，造成整個紀錄功能卡死。

## Goals / Non-Goals

**Goals:**
- **自動修補歷史 Session**：在 `MaybeReconcile` 時（如 `tt report` 或 `tt serve` 啟動時），自動偵測並填補缺失 `project` 或 `model` 欄位的 session。
- **懸空 Turn 逾時強制關閉**：在 `reconcile` 流程中，即使 dangling turn 的 process 仍存活，若其已超過 `idle-threshold`（15 分鐘），則強制將其關閉並計算 token。
- **防止 RecordPrompt 卡死**：在 `RecordPrompt` 中，若有已逾期的 active turn，先將其強制關閉再建立新 turn。

**Non-Goals:**
- 不對現有的資料庫 Schema 進行變更（沿用既有的 `sessions` 與 `turns` 表欄位）。
- 不改變其他非 `antigravity` 工具（如 `claude-code`, `copilot-cli`）的既有行為，除非其也受到進程存活超時邏輯的正面影響。

## Decisions

### 1. 歷史 Session 背景自動修補 (repairSessions)
- **作法**：在 `reconcile` 開始時，執行 `repairSessions(db)`：
  1. 查詢所有 `project IS NULL OR project = '' OR model IS NULL OR model = ''` 的 sessions。
  2. 對於每個匹配的 session，找出其所屬的第一個具有 `transcript_path` 且檔案存在的 turn。
  3. 讀取 `transcript_full.jsonl`，遞迴掃描 JSON 結構中的字串，尋找屬於使用者 Home 目錄的絕對路徑，但過濾掉系統/工具路徑（如 `.gemini`、`.claude`、`.copilot`、`Library`、`Downloads`、`Desktop`、`Applications`）。
  4. 從尋找到的路徑，向上層目錄遞迴尋找 `.git` 或 `go.mod` 來重構專案根目錄；若均無則 fallback 至 `os.Getwd()`。
  5. 若 `model` 欄位為空，則從日誌中解析最尾端的 assistant model，或從 settings.json 載入，或 fallback 至 `gemini-3.5-flash`。
  6. 更新該 session 的 `project` 與 `model` 欄位。

### 2. 進程存活超時自動關閉 (idle-threshold)
- **作法**：在 `reconcile.go` 中，將 `idle-threshold` 定義為 15 分鐘。
- 在 `reconcileTurn` 中，將判定進程存活而 skip 的邏輯修正為：
  - 若 `isLatest && process.IsAlive(...) && time.Since(dt.promptAt) <= 15*time.Minute` 時才 return nil。
  - 若時間已超過 15 分鐘，則即使進程仍存活，也強制執行 token 提取與 `UPDATE turns SET response_at = ...`。

### 3. RecordPrompt 自動搶佔逾期懸空 Turn
- **作法**：在 `RecordPrompt` 中，針對 `antigravity` 工具偵測 active turn 的邏輯修正為：
  - 查詢目前 session 的 active turn。
  - 若 active turn 存在，解析其 `prompt_at` 時間：
    - 若 `time.Since(promptAt) > 15*time.Minute`，則在 DB 中將該 dangling turn 強制關閉（`UPDATE turns SET response_at = ? WHERE id = ?`），以釋放鎖定，接著順利建立新 turn。
    - 若未逾期，則維持原樣直接 return nil。

## Risks / Trade-offs

- **[Risk]** 遞迴掃描 JSON 日誌檔案可能在極端大檔案下影響效能。
  - **Mitigation**：此修補邏輯僅在 session missing 欄位時執行一次，修補後欄位填滿便不再進入該邏輯。且掃描到第一個有效的專案路徑後即停止該 session 的掃描。
- **[Risk]** 使用者可能在背景進行長時間（超過 15 分鐘）的單次 turn 運算（例如極為複雜的程式生成）。
  - **Mitigation**：15 分鐘對於一般 AI tool 互動而言已非常足夠，若真的超過且被強制關閉，在使用者下一次發送 prompt 時會順利啟動新 turn，不影響後續紀錄的完整性。
