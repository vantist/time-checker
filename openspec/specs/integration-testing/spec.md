# integration-testing Specification

## Purpose
TBD - created by archiving change integration-testing-unpushed-features. Update Purpose after archive.
## Requirements
### Requirement: 整合測試環境隔離與臨時二進位檔編譯
整合測試套件 MUST 在執行測試前動態編譯出一個臨時的 `tt` 二進位檔。
在執行該二進位檔時，測試套件 SHALL 透過設定環境變數 `HOME` 與 `TT_DB_PATH` 隔離執行環境，確保測試不會影響本機環境。

#### Scenario: 隔離執行 tt 命令
- **WHEN** 整合測試套件啟動並透過 `runTT` 輔助函式執行臨時 `tt`
- **THEN** 系統 SHALL 於暫存目錄下讀寫獨立的 SQLite 資料庫檔，且不污染真實的使用者環境。

### Requirement: 驗證 Git 分支自動修復與 Reconcile
整合測試 SHALL 驗證在 SQLite 資料庫中缺少 Git 分支名稱的 session，能被 `reconcile` 機制自動修復。

#### Scenario: 整合測試自動修復無 Git 分支資訊的 Session
- **WHEN** 在 SQLite 中手動寫入一筆無分支名稱的 session，並透過 CLI 執行 `reconcile` 命令
- **THEN** 系統 SHALL 自動取得當前 Git 分支，並修復該 session 的 branch 欄位。

### Requirement: 驗證前一次 Active Turn 自動 Pre-empt 關閉
整合測試 SHALL 驗證當一個 active turn 尚未關閉時，新發起的 turn 會主動 preempt（搶占）並關閉前一個 turn。

#### Scenario: 連續錄製 prompt 觸發 preempt
- **WHEN** 連續執行兩次 `tt` 的 prompt 錄製，且第一次錄製並未發送結束（Stop）事件
- **THEN** 系統 SHALL 自動將第一次 turn 進行 preempt，將其結束時間寫入並更新為 active = 0。

### Requirement: 驗證 15 分鐘空閒超時自動 Reconcile
整合測試 SHALL 驗證對於建立時間超過 15 分鐘且無結束時間的 dangling turn，系統會以 15 分鐘空閒時間為上限進行自動 reconcile 關閉。

#### Scenario: 超時懸空 turn 自動關閉
- **WHEN** 在 SQLite 中寫入一個開始時間為 20 分鐘前且 `response_at` 為空的 dangling turn，並執行 `reconcile`
- **THEN** 系統 SHALL 自動將其結束時間設為開始時間加上 15 分鐘，重算 token 並將 active 設為 0。

### Requirement: 驗證多工具 Stdin 與 Log 解析
整合測試 SHALL 模擬 Claude Code、Copilot CLI 與 Google Antigravity 的 stdin JSON 格式與對應的 transcript / events 日誌檔案，驗證 `tt` 解析 token 與 model 資訊並儲存至 SQLite 的行為。

#### Scenario: 解析不同工具的 stdin payload 與日誌
- **WHEN** 分別提供 Claude Code, Copilot CLI, Google Antigravity 的 stdin，以及模擬的 transcript.jsonl / events.jsonl 日誌，並呼叫錄製 CLI
- **THEN** 系統 SHALL 正確解析日誌中記載的 model 名稱與 token 使用量（含 input/output tokens），並寫入對應的 SQLite 資料表中。

