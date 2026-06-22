## Context

系統採用 `(process_pid, process_start)` 作為穩定工作 Session 的唯一鍵。然而，當 Claude Code 執行 `/clear`（清空歷史紀錄）或使用 `claude --resume` 重新連接進程與對話時，Conversation ID（Session UUID）或進程的 PID/啟動時間會發生變化。為了確保系統在此類情境下依然能正確累計時間與 Token，且不發生資料衝突或遺失，本設計梳理並規範了整個系統（資料庫、記錄器、Transcript 解析與 Reconcile 層、以及 Antigravity 整合去重邏輯）的具體行為架構。

## Goals / Non-Goals

**Goals:**
- 規範 `/clear` 觸發時的運作行為，包括 Session 級別對齊、Turn 級別隔離、Reconcile 邊界判斷與 Clear Race 補救。
- 規範 `--resume` 觸發時的運作行為，包括透過 `conversation_id` 匹配現有 Session、更新 Process Key 並復原。
- 規範 Antigravity 特殊的去重與修補機制（單一 Turn 與 Clear Race 補救）。
- 確保所有行為皆有對應的單元測試覆蓋（包含 `session_test.go`、`extract_test.go` 與 `recorder_test.go`）。

**Non-Goals:**
- 不對現有的資料庫 Schema 進行破壞性變更。
- 不修改 CLI hooks 的靜默失敗行為（必須維持 exit 0）。
- 不變更時間統計（Aggregation）的核心最大閾值截斷邏輯。

## Decisions

### 1. Session 級別對齊與 `--resume` 處理 (`db.UpsertSession`)
- **決策：**
  - 在 `db.UpsertSession` 中以 `(process_pid, process_start)` 為主鍵（Process Key）。
  - **對於 `/clear`：** 由於 PID 與啟動時間不變，僅 `session_id` (即 `conversation_id`) 改變，`upsertByProcessKey` 會回傳最初建立的 `stableID`，使所有 Turn 始終對齊在同一個資料庫 Session。
  - **對於 `--resume`：** 由於進程重開，PID 與啟動時間均變更，但 `conversation_id` 相同。當無法以 Process Key 找到 Session 時，系統應改用 `conversation_id` 尋找現有記錄。若成功匹配，則將該記錄之 `process_pid` 與 `process_start` 更新為新進程的值，並回傳原 Session ID。
- **替代方案：** 每次都建立新 Session。這會導致同一個工作進程的時間統計分裂為多筆記錄，失去工作時間追蹤的連續性。

### 2. Turn 隔離與去重邏輯 (`recorder.RecordPrompt`)
- **決策：**
  - 為了兼容 Antigravity 在 LLM Agent 思考步驟中多次觸發 PreInvocation 的特性，`RecordPrompt` 在寫入 Turn 時需檢查有無 active turn（`response_at IS NULL` 的 Turn 數）。若 `activeCount > 0`，則去重不重複插入。
  - 當使用者執行 `/clear` 並提交新 Prompt 時，由於前一個 Turn 在結束時已由 `RecordResponse` 正常關閉（`response_at` 已寫入），`activeCount` 為 0，因此能順利建立新 Turn 達成隔離，不會產生資料污染。

### 3. Reconcile 邊界判斷與 Clear Race 補救
- **決策：**
  - **Reconcile 邊界：** 由於 `/clear` 會生成新日誌路徑，Reconcile 時會因為 `dt.nextTranscriptPath != dt.transcriptPath` 將 `toOffset` 設為 `-1`（意即讀取至舊日誌檔的尾端），防止越界讀取。
  - **Clear Race 補救：** 當 Stop 鉤子在 `/clear` 之後立刻被觸發時，Transcript 提取層的 `ExtractLastTurn` 偵測到最後一筆 user entry 沒有 assistant entry 回覆，會自動 Fallback 讀取上一個 Assistant 答覆窗口，確保舊數據不被覆蓋或遺失。

## Risks / Trade-offs

- **[Risk]** 進程 PID 衝突（PID reuse）導致 session 被錯誤對齊。
  - **[Mitigation]** 藉由使用 `(process_pid, process_start)` 組合鍵，僅當同一個 PID 在同一秒鐘內啟動時才會衝突，機率極低。
