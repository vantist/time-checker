## Why

目前 `tt setup` 有兩個主要限制：
1. **不支援多工具同時設定**：目前的程式碼採用多個獨立的 `if ... return nil` 區塊，若傳入 `--claude-code --copilot`，僅有第一個比對成功的工具（如 Claude Code）會被設定，其後工具會被直接略過。
2. **預設（無參數時）不會執行任何設定**：無參數執行時僅會印出 Help 說明，無法快速上手。

## What Changes

改善 `tt setup` 的運作邏輯：
1. **多工具並行設定**：依序執行被選中的多個工具設定，不再提早 return。
2. **預設智慧偵測行為**：在未傳入任何 flag 時，自動偵測使用者家目錄（`HOME`）下是否存在各 AI 工具的設定主目錄（`~/.claude`、`~/.copilot`、`~/.gemini`、`~/.codex`）。若存在，則預設設定該工具。
3. **無偵測時提示**：若未帶 flag 且未偵測到任何適用工具，輸出友善的提示訊息。

## Capabilities

### New Capabilities

- `setup-improvements`: 提升 `tt setup` 指令的實用性，包含多工具並行設定、智慧自動偵測 AI 工具目錄並自動安裝，以及無偵測工具時的提示。

### Modified Capabilities

(none)

## Impact

- Affected specs:
  - `openspec/specs/setup-improvements/spec.md`
- Affected code:
  - New: (none)
  - Modified:
    - `cmd/tt/setup_cmd.go`
    - `cmd/tt/setup_cmd_test.go`
    - `internal/setup/setup.go`
    - `internal/setup/setup_test.go`
  - Removed: (none)

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-20-brainstorm-setup-cmd-improvements.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
