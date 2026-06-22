## Context

目前 `tt setup` 僅支援 `Claude Code` 與 `GitHub Copilot CLI` 的 hook/指令設定。然而，底層與 log 解析實作已能支援 `Google Antigravity` 與 `OpenAI Codex`，這造成了設定面與底層實作能力的不一致，應予擴充。

## Goals / Non-Goals

**Goals:**
- 提供 `tt setup --antigravity` 與 `tt setup --codex` CLI 參數以供自動安裝與配置對應 hooks。
- 透過通用重構 Helper `mergeHooksFile` 來維護 hook 設定檔的合併、篩選 `_owner == "tt"` 以及冪等寫入邏輯。
- 支援在 `tt record` 中解析來自 stdin 的 Antigravity 專屬欄位（`conversationId` 與 `transcriptPath`），並將其正確對應至 session 的 session ID 與 transcript 路徑。

**Non-Goals:**
- 不支援非 JSON 格式的 hooks 設定。
- 不在本變更中支援除了 Antigravity 與 Codex 之外的其他工具。
- 不變更資料庫 schema，只在寫入 Log 的 payload 對應中進行欄位正規化。

## Decisions

### 1. 抽取 `mergeHooksFile` Helper
- **方案**：在 `internal/setup/setup.go` 中，實作一個通用函式 `mergeHooksFile(configPath string, defaultOwner string, hookExtractor func([]byte) (map[string]interface{}, error), hookMerger func(map[string]interface{}) (map[string]interface{}, error)) error` 或類似的輔助函式。
- **原因**：Claude Code、Antigravity 與 Codex 均採用 JSON 結構作為 Hook 設定，且具備類似的冪等清理邏輯（清理舊的 `_owner == "tt"` 項目並插入新版本），以及要求 `0o600` 權限寫入。抽取 Helper 可減少重複程式碼，提升維護性。

### 2. Antigravity 與 Codex Hook 配置結構
- **Antigravity Hook 結構 (`~/.gemini/config/hooks.json`)**：
  ```json
  {
    "tt": {
      "PreInvocation": [
        {
          "_owner": "tt",
          "type": "command",
          "command": "tt record prompt --tool antigravity"
        }
      ],
      "Stop": [
        {
          "_owner": "tt",
          "type": "command",
          "command": "tt record response --tool antigravity"
        }
      ]
    }
  }
  ```
- **Codex Hook 結構 (`~/.codex/hooks.json`)**：
  ```json
  {
    "hooks": {
      "UserPromptSubmit": [
        {
          "_owner": "tt",
          "type": "command",
          "command": "tt record prompt --tool codex"
        }
      ],
      "Stop": [
        {
          "_owner": "tt",
          "type": "command",
          "command": "tt record response --tool codex"
        }
      ]
    }
  }
  ```
- **原因**：與該等 AI 工具的原生 hooks 設計相容，並在 hook payload 中透過 `--tool` 明確指定來源。

### 3. Record Stdin 欄位對應
- **方案**：擴充 `cmd/tt/record.go` 中的 `hookPayload` struct，新增欄位 `ConversationID string json:"conversationId,omitempty"` 與 `TranscriptPath string json:"transcriptPath,omitempty"`。
- **解析邏輯**：在 `readStdinJSON()` 中，若傳入的 `tool` 為 `antigravity`，且 `ConversationID` 或 `TranscriptPath` 有值，則分別賦值給 `SessionID` 與 `TranscriptPath`。
- **原因**：Antigravity hooks 透過 stdin 傳遞 JSON Payload 時的屬性鍵值與 tt 內部不同，此對應層可確保底層儲存與統計邏輯能無縫重用。

## Risks / Trade-offs

- **[Risk] 設定檔目錄不存在** → **[Mitigation]** 在寫入前使用 `os.MkdirAll` 搭配 `0o700` 權限確保父目錄已被建立。
- **[Risk] 設定檔 JSON 解析失敗** → **[Mitigation]** 若原檔案為非 JSON 或損毀，應備份原檔或將其視為空配置覆寫，並記錄 error。
