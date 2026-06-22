## ADDED Requirements

### Requirement: SetupCommandMultiToolSetup
系統 SHALL 支援在執行 `tt setup` 時帶有複數個標記（例如 `--claude-code` 與 `--copilot`），且系統 SHALL 依序完成所有被選取之 AI 工具的 hook 設定，而不因其中一個設定成功而提早結束。

#### Scenario: 同時設定多個工具的 hooks
- **WHEN** 呼叫 `tt setup --claude-code --copilot` 且皆執行成功
- **THEN** stdout 依序輸出 `Claude Code hooks configured in ~/.claude/settings.json` 與 `GitHub Copilot CLI hooks configured in ~/.copilot/hooks/tt.json`
- **THEN** 同時更新 `~/.claude/settings.json` 與 `~/.copilot/hooks/tt.json`

### Requirement: SetupCommandAutoDetection
系統在執行 `tt setup` 且未帶有任何 flag 時，SHALL 自動偵測使用者家目錄（`HOME`）下是否存在各 AI 工具的設定目錄。若偵測到對應目錄存在，系統 SHALL 自動為該工具設定 hooks，不需使用者手動傳入 flag。

具體偵測規則如下：
- 若存在 `~/.claude` 目錄，則自動設定 Claude Code hooks
- 若存在 `~/.copilot` 目錄，則自動設定 GitHub Copilot CLI hooks
- 若存在 `~/.gemini` 目錄，則自動設定 Google Antigravity hooks
- 若存在 `~/.codex` 目錄，則自動設定 OpenAI Codex hooks

#### Scenario: 自動偵測到部分工具存在並設定
- **WHEN** 呼叫 `tt setup`（無參數），且家目錄下僅存在 `~/.claude` 與 `~/.gemini` 目錄
- **THEN** 系統自動設定 Claude Code 與 Google Antigravity 的 hooks
- **THEN** stdout 輸出 `Claude Code hooks configured in ~/.claude/settings.json` 與 `Google Antigravity hooks configured in ~/.gemini/config/hooks.json`

### Requirement: SetupCommandNoToolWarning
系統在執行 `tt setup` 且未帶有任何 flag 時，若未偵測到任何適用工具的設定目錄，SHALL 輸出友善提示訊息並結束，SHALL NOT 修改任何檔案。

#### Scenario: 未偵測到任何適用工具時輸出提示
- **WHEN** 呼叫 `tt setup`（無參數），且家目錄下不存在 `~/.claude`, `~/.copilot`, `~/.gemini`, `~/.codex` 中的任何一個目錄
- **THEN** stdout 輸出 `No supported AI tools detected...` 提示訊息
