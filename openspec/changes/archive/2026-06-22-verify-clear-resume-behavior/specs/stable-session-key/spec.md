## MODIFIED Requirements

### Requirement: 穩定工作 session 識別

系統 SHALL 以 `(process_pid, process_start)` 組合作為工作 session 的唯一識別符。同一 Claude Code 進程內，無論執行多少次 `/clear`，皆 SHALL 對應到同一個 session 記錄。此外，當使用 `--resume` 重開進程時，系統 SHALL 透過對齊 `conversation_id` 復原 Session，並更新其 `(process_pid, process_start)` 為新進程的值。

#### Scenario: 首次建立工作 session

- **WHEN** `tt record prompt` 收到 `$PROCESS_PID` 與 `$PROCESS_START`，且 DB 中不存在相同 `(process_pid, process_start)` 的 session
- **THEN** 建立新 session 記錄，`process_pid` 與 `process_start` 設為收到的值，`conversation_id` 設為 stdin 的 `session_id` UUID

#### Scenario: /clear 後繼續記錄同一工作 session

- **WHEN** `tt record prompt` 收到相同的 `$PROCESS_PID` 與 `$PROCESS_START`，但 `session_id` UUID 與現有 session 的 `conversation_id` 不同（代表發生過 `/clear`）
- **THEN** 更新現有 session 的 `conversation_id` 為新 UUID，`last_seen` 更新為當前時間，不建立新 session

#### Scenario: 舊資料（無 process_pid）不受影響

- **WHEN** DB 中存在 `process_pid = NULL` 的 session 記錄
- **THEN** 這些記錄 SHALL 繼續可讀，不被修改 or 刪除

#### Scenario: 透過 --resume 重開進程並復原 session

- **WHEN** `tt record prompt` 收到不同的 `$PROCESS_PID` 與 `$PROCESS_START`，但其 `session_id` (即 `conversation_id`) 與 DB 中現有 session 記錄相同
- **THEN** 系統 SHALL 將該 session 記錄之 `process_pid` 與 `process_start` 更新為收到的新進程值，並回傳原 session ID 以利後續 Turn 對齊

## ADDED Requirements

### Requirement: Turn 級別的隔離與去重

當 `/clear` 被執行時，上一筆對話已正常關閉，因此新一筆對話會被建立為獨立的新 Turn，兩者完全獨立。而對於 Antigravity 的 PreInvocation 觸發，系統 SHALL 實作 `activeCount > 0` 則不重複插入的去重邏輯，將一整個對話命令合併成單一 Turn。

#### Scenario: /clear 後的獨立新 Turn

- **WHEN** 執行 `/clear` 後，前一個 Turn 的 `response_at` 欄位已被寫入關閉，接著執行 `RecordPrompt`
- **THEN** 系統確認當前已無 active turn，因而正常插入一筆全新 Turn

#### Scenario: Antigravity 多步思考去重合併為單一 Turn

- **WHEN** 在同一個 LLM Agent 思考步驟中多次觸發 PreInvocation，且當前已有 active turn（`response_at IS NULL` 的 Turn 數大於 0）
- **THEN** 系統去重不重複插入，使整筆對話合併為單一 Turn

### Requirement: Reconcile 與日誌路徑切換邊界處理

當使用者執行 `/clear` 導致日誌路徑切換或日誌截斷時，Reconcile 時系統 SHALL 藉由比較 `dt.nextTranscriptPath != dt.transcriptPath` 將 `toOffset` 設為 `-1`（意即讀取至舊日誌檔的尾端），以防止越界讀取。

#### Scenario: /clear 導致日誌路徑變更時 Reconcile 限制讀取邊界

- **WHEN** 進行 Reconcile 且偵測到 `nextTranscriptPath` 與當前 `transcriptPath` 不同
- **THEN** 系統將 `toOffset` 設為 `-1`（讀取至舊日誌檔尾端），避免越界讀取

### Requirement: Clear Race 補救機制

若 Stop 鉤子在 `/clear` 之後立刻被觸發時，系統的 Transcript 提取層偵測到最後一筆 user entry 沒有 assistant entry 回覆，SHALL 自動 Fallback 讀取上一個 Assistant 答覆窗口，確保舊資料不被覆蓋或遺失。

#### Scenario: Stop 觸發時發生 Clear Race 的 Fallback 讀取

- **WHEN** Stop 鉤子觸發且偵測到最後一筆 user entry 沒有對應的 assistant entry 回覆
- **THEN** `ExtractLastTurn` 自動 Fallback 讀取上一個完整的 Assistant 答覆窗口，確保資料完整寫入而不被覆蓋
