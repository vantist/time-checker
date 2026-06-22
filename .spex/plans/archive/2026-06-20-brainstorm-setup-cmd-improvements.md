# setup-cmd-improvements

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

目前 `tt setup` 有兩個限制：
1. **不支援多工具同時設定**：目前的程式碼採用多個獨立的 `if ... return nil` 區塊，若傳入 `--claude-code --copilot`，只有 Claude Code 會被設定，其後工具會被直接略過。
2. **預設（無參數時）不會執行任何設定**：無參數執行時僅會印出 Help 說明，無法快速上手。

## Decision

改善 `tt setup` 的運作邏輯：
1. **多工具並行設定**：依序執行被選中的多個工具設定，不再提早 return。
2. **預設智慧偵測行為**：在未傳入任何 flag 時，自動偵測使用者家目錄（`HOME`）下是否存在各 AI 工具的設定主目錄（`~/.claude`、`~/.copilot`、`~/.gemini`、`~/.codex`）。若存在，則預設設定該工具。
3. **無偵測時提示**：若未帶 flag 且未偵測到任何適用工具，輸出友善的提示訊息。

## Rationale

1. **使用者體驗提升**：在全新的工作環境中，使用者只需要輸入 `tt setup` 即可依據現有安裝好的工具自動配置 hooks，不必記憶複雜的 flag。
2. **靈活性**：同時支援以 explicit flags 指定要安裝哪些工具的 hooks（例如 `tt setup --claude-code --copilot`），不會影響進階用戶。
3. **安全且乾淨**：不主動建立未使用 AI 工具的空目錄，避免造成檔案系統污染（Option 2 的選擇）。

## Approach

1. 在 `internal/setup/setup.go` 中新增偵測函數：
   - `IsClaudeCodeActive() bool` (檢查 `~/.claude`)
   - `IsCopilotActive() bool` (檢查 `~/.copilot`)
   - `IsAntigravityActive() bool` (檢查 `~/.gemini`)
   - `IsCodexActive() bool` (檢查 `~/.codex`)
2. 修改 `cmd/tt/setup_cmd.go` 中的 `RunE` 邏輯：
   - 當沒有任何 flag 被選取時，自動執行上述偵測函數並將對應的 boolean 設為 true。
   - 遍歷所有選定工具，執行安裝。
   - 若最後未設定任何工具，印出提示訊息。
3. 在 `cmd/tt/setup_cmd_test.go` 中新增測試案例，驗證：
   - 當特定設定目錄存在時，`tt setup`（無參數）會正確自動設定該工具。
   - 當沒有任何設定目錄存在時，`tt setup`（無參數）會輸出 `No supported AI tools detected...` 的提示訊息。
   - 傳入多個 flag（如 `--claude-code --copilot`）時，會順利對兩個工具都進行設定。

## Design Notes

### 執行流程偽代碼

```go
hasUserFlags := claudeCode || copilot || antigravity || codex

if !hasUserFlags {
    if IsClaudeCodeActive() { claudeCode = true }
    if IsCopilotActive() { copilot = true }
    ...
}

configured := false
if claudeCode {
    setup.SetupClaudeCode()
    configured = true
}
if copilot {
    setup.SetupCopilot()
    configured = true
}
...
if !configured {
    fmt.Println("No supported AI tools detected...")
}
```

## Open Questions

（無，設計已收斂）
