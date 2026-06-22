## Context

本地已有多個未推送的實作功能（包括分支修復、主動 pre-empt 搶占、15 分鐘空閒超時以及多工具 log 提取與 fallback）。為了驗證這些功能在真實 CLI 呼叫、環境變數隔離和 SQLite 資料庫層面是否正確動作，我們需要設計並實作一個 Go 的端到端整合測試套件。

## Goals / Non-Goals

**Goals:**
- 實作動態編譯臨時 `tt` 二進位檔的測試腳手架，確保測試間環境變數（如 `HOME`、`TT_DB_PATH`）與 SQLite 檔案的完整隔離。
- 模擬並驗證多個工具（Claude Code, Copilot CLI, Google Antigravity）各自不同的 stdin JSON 與 log 檔案的端到端解析與儲存。
- 驗證 Git 分支自動修復、主動 pre-empt 搶占、15 分鐘空閒超時、以及 fallback 預設模型等行為。

**Non-Goals:**
- 不對 Web 儀表板（Web Dashboard）進行整合測試。
- 不對真實外部網路或第三方服務（例如真實的 Claude API 連線）進行整合，所有輸入和日誌均使用 Mock 檔案與 Stdin 模擬。

## Decisions

### 1. 子進程命令列執行法（Black-Box Integration Test）
- **說明**：測試執行時會先編譯出臨時 `tt` 二進位檔，並在執行 CLI 時完全覆寫 `HOME`、`TT_DB_PATH` 以實現環境隔離與行為模擬。
- **理由**：子進程執行法能完整模擬 stdin 讀取、進程生命週期判斷（IsAlive）以及 CLI 啟動時的檔案鎖（File Lock）與初始化邏輯，並避免 Cobra commands 全域 flag 殘留導致的測試干擾。
- **替代方案**：直接在記憶體中呼叫 `cmd.Execute()`。缺點是難以模擬完整的 stdin 流和環境變數隔離。

### 2. SQLite 資料庫層級驗證
- **說明**：在 CLI 執行結束後，直接透過 Go `database/sql` 驅動讀取臨時的 SQLite 檔案，對 `sessions` 與 `turns` 資料表進行斷言驗證。
- **理由**：相較於比對 CLI 的 stdout 輸出，直接驗證 DB schema 中的資料結構更為穩定，且能精確驗證 pre-empted turn 的結束時間、重算後的 token 數量等資料。
- **替代方案**：透過 `tt report` 的輸出來驗證。缺點是報告輸出格式容易改變，且無法精確確認底層欄位細節。

### 3. 多工具 Mock 紀錄檔格式與 Stdin Payload
- **說明**：在測試中寫入符合特定工具結構的暫存檔案（如 Claude Code transcript.jsonl，Copilot events.jsonl，Google Antigravity transcript），以模擬不同 AI 工具的執行場景。
- **理由**：確保不同工具的 parser 能正確在 integration 測試中執行並解析出對應的 token 與 model 資訊。

## Risks / Trade-offs

- **[Risk]** 每次測試執行 `go build` 會增加測試耗時。
  - **[Mitigation]** 在 `TestMain` 中只執行一次 `go build` 編譯出臨時 `tt` 二進位檔，並在所有整合測試案例中複用該檔案，最後在測試結束後統一清理。
- **[Risk]** 臨時檔案殘留。
  - **[Mitigation]** 使用 Go 測試框架的 `t.TempDir()` 或 `t.Cleanup` 機制自動管理，確保不論測試成功或失敗，所有產生的暫存目錄與 SQLite 檔案都會被自動清除。
