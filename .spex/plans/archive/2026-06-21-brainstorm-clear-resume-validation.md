# Verify /clear and --resume Behavior

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

使用者提出疑問，希望確認系統中 `/clear`（清除歷史紀錄）與 `--resume`（重新關聯進程/Resume 對話）在整個架構中的運作行為是否有被正確實作，以及是否會產生資料衝突或時間統計上的異常。

## Decision

經由對程式碼與單元測試的審閱與分析，確定系統現行架構已針對 `/clear` 與 `resume` 設計了相應的防護與補救機制，運作行為完全符合預期。

## Rationale

系統透過兩大防線來確保清空與恢復行為的正確性：
1. **資料庫 Process Key 隔離與對齊** (`db.UpsertSession`)：利用 PID 與啟動時間作為 Process 唯一鍵，完美處理在同一個進程內多次 `/clear` 生成新會話 ID 但對齊到同一個 Session 主鍵，或者重開進程帶入 `--resume` 時的 Session 復原。
2. **Reconcile 層與 Transcript 解析的隔離** (`ExtractLastTurn` & `reconcileTurn`)：針對 `/clear` 造成的日誌路徑切換或日誌截斷，Transcript 提取層使用路徑邊界判斷隔離不同 Turn，且有專屬的 `ClearRace` 補救機制以防止 Tokens 被記錄為 0 或漏記。

## Approach

進行全面的架構審查與原理說明，對齊以下三種情境的運作流程：
- 情境一：互動中執行 `/clear` 繼續對話
- 情境二：透過 `--resume` 重新連接進程與對話
- 情境三：Antigravity 特殊的去重與修補機制

## Design Notes

### 1. `/clear` 後繼續的行為機制
當使用者執行 `/clear` 時，底層行為如下：
- **Session 級別的對齊**：
  - `/clear` 會使 CLI 產生一個新的 Conversation ID（代表新的 `SessionID`），但因為 PID 與啟動時間（Process Key）不變，`db.UpsertSession` 中的 `upsertByProcessKey` 會回傳最初建立的 `stableID`，使所有 Turn 始終綁定在同一個資料庫 Session。
- **Turn 級別的隔離**：
  - 前一個 Turn 在執行結束時，其 `response_at` 已被 `RecordResponse` 正常填入（已關閉）。
  - 當使用者輸入 `/clear` 並提交新 Prompt 時，`RecordPrompt` 會確認當前無未關閉的 active turn（`response_at IS NULL` 的 Turn 數為 0），因而正常插入一筆全新 Turn。
- **Reconcile 的邊界判斷**：
  - 由於 `/clear` 會生成新日誌路徑，Reconcile 時會因為 `dt.nextTranscriptPath != dt.transcriptPath` 將 `toOffset` 設為 `-1`（意即讀取至舊日誌檔的尾端），防止越界讀取。
- **Clear Race 補救**：
  - 若 Stop 鉤子在 `/clear` 之後立刻被觸發時，`ExtractLastTurn` 偵測到最後一筆 user entry 沒有 assistant entry 回覆，會自動 Fallback 讀取上一個 Assistant 答覆窗口，確保舊資料不會被覆蓋或遺失。

### 2. 進程 Resume 的行為機制 (如 `claude --resume`)
當進程被關閉、重開並指定 `--resume` 時：
- **Process Key 更新**：
  - 重開進程代表 Process PID 與啟動時間都變更了，但 `conversation_id` 依然相同。
  - `upsertByProcessKey` 會因為無法以 Process Key 找到 Session，而改用 `ConversationID` 進行 resume 檢查。
  - 當成功匹配到 `conversation_id` 時，它會將該 Session 資料庫記錄的 `process_pid` 與 `process_start` 更新為新進程的值，並回傳原有的 Session ID。
- **後續行為**：
  - 後續提交的 prompt 與 response 會藉由更新後的進程鍵對齊回原 Session，並繼續在原 Session 中新增 Turn，完美復原。

### 3. Antigravity 整合特性
- **單一 Turn 機制**：
  - 由於 Antigravity 的 PreInvocation 會在每個 LLM Agent 思考步驟中觸發，我們實作了 `activeCount > 0` 則不重複插入的 deduplication 邏輯，將一整個對話命令合併成單一 Turn。
  - 當 `/clear` 被執行時，上一筆對話已正常關閉，因此新一筆對話會被建立為獨立的新 Turn，兩者完全獨立，不會有任何資料相互污染。

## Insights to Capture

- `internal/db/session_test.go`: `TestUpsertSession_Resume` 驗證 resume 行為
- `internal/transcript/extract_test.go`: `TestExtractLastTurn_ClearRace` 驗證 clear race 補救機制
- `internal/recorder/recorder_test.go`: `TestRecordPrompt_StableSession` 驗證 clear 後 Session 複用與 Turn 隔離

## Open Questions

(none - 架構行為已確認符合預期)
