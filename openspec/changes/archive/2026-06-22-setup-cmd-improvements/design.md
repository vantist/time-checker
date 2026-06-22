## Context

目前的 `tt setup` 存在兩個局限性：
1. **多工具設定互斥**：在 `cmd/tt/setup_cmd.go` 中，設定邏輯採用多個獨立的 `if ... return nil` 區塊。這導致如果使用者同時傳入 `--claude-code` 與 `--copilot`，只有第一個比對成功的工具（如 Claude Code）會被設定，其後工具的設定則會被跳過。
2. **預設無行為**：在未帶有任何 flag 執行 `tt setup` 時，指令只會顯示說明訊息（`cmd.Help()`），無法自動化執行任何設定，對新使用者來說不夠直覺。

## Goals / Non-Goals

**Goals:**
* 支援同時傳入多個 flag 進行多個 AI 工具的 hook 設定。
* 當 `tt setup` 未帶任何 flag 執行時，自動偵測使用者家目錄（`HOME`）下是否存在對應工具的設定目錄（`~/.claude`、`~/.copilot`、`~/.gemini`、`~/.codex`）。如果偵測到，則自動為該工具進行設定。
* 若未傳入 flag 且未偵測到任何適用工具，輸出友善的提示訊息 `No supported AI tools detected...`。

**Non-Goals:**
* 本變更不會在自動偵測時主動建立未安裝/不存在之 AI 工具的空目錄。
* 本變更不涉及修改現有各工具 hook 的具體 JSON 設定內容或寫入邏輯。

## Decisions

### 1. 智慧偵測函數的設計與放置

在 `internal/setup/setup.go` 中新增四個導出函數，用於檢查各工具的設定目錄是否存在：
* `IsClaudeCodeActive() bool`：檢查 `~/.claude` 是否存在。
* `IsCopilotActive() bool`：檢查 `~/.copilot` 是否存在。
* `IsAntigravityActive() bool`：檢查 `~/.gemini` 是否存在。
* `IsCodexActive() bool`：檢查 `~/.codex` 是否存在。

**決策原因：**
將偵測邏輯放在 `internal/setup` 套件中，可使 `cmd/tt` 保持簡潔，並讓偵測邏輯便於在單元測試中被模擬。

### 2. 測試中的家目錄（Home Directory）模擬

在單元測試 `internal/setup/setup_test.go` 與 `cmd/tt/setup_cmd_test.go` 中，將透過 Go 測試標準庫的 `t.Setenv("HOME", tempDir)` 暫時修改環境變數 `HOME` 指向臨時目錄，以驗證不同目錄存在狀況下的偵測行為。

**決策原因：**
使用環境變數重定向是 Go 測試中模擬家目錄最無侵入性且乾淨的做法，且 `t.Setenv` 在測試結束時會自動恢復環境變數。

### 3. 指令執行邏輯重構

修改 `cmd/tt/setup_cmd.go` 中的 `RunE` 邏輯：
* 先讀取所有 flags。
* 若所有 flags 皆為 false，則依序呼叫 `IsClaudeCodeActive()`、`IsCopilotActive()`、`IsAntigravityActive()`、`IsCodexActive()`，並將對應的變數設為 true。
* 依序檢查各變數：
  * 若為 true，則執行對應的設定函數（如 `setup.SetupClaudeCode()`），並記錄已設定的狀態。
  * 若執行過程中遇到錯誤，則立即中斷並返回錯誤。
* 若最後沒有任何工具被設定（且原先沒有手動傳入 flag），則輸出 `No supported AI tools detected...`。

## Risks / Trade-offs

* **[Risk]** 在測試中修改 `HOME` 環境變數若未正確隔離，可能影響其他測試。
  * **[Mitigation]** 僅在子測試（subtests）中使用 `t.Setenv`。Go 的 `t.Setenv` 具有 thread-safe 及自動清理機制，能安全地將變更隔離在該測試函數內。
* **[Risk]** 使用者家目錄下的工具目錄可能是檔案而非目錄，導致 `Stat` 誤判。
  * **[Mitigation]** 偵測函數應使用 `os.Stat` 取得檔案資訊，並利用 `info.IsDir()` 確保其為目錄，而非僅檢查 `ErrNotExist`。
