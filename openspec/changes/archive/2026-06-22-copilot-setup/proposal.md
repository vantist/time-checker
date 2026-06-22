## Why

目前 `tt setup --copilot` 僅會印出指示說明，要求使用者手動新增與修改 `~/.copilot/settings.json`。然而，這項指示說明在兩個地方有落差：
1. **設定路徑錯誤**：GitHub Copilot CLI 的使用者級 hooks 設定檔實際應位於 `~/.copilot/hooks/` 目錄下的任何 JSON 檔案（例如 `~/.copilot/hooks/tt.json`）。
2. **JSON 格式錯誤**：Copilot CLI 載入的 hooks 格式並非扁平的 Key-Value，而是帶有 `"version": 1` 且各事件（`userPromptSubmitted`、`agentStop`）對應 `type: "command"` 物件陣列的格式。

同時，其他 AI 工具（Claude Code, Antigravity, Codex）的 setup 都已實現冪等（Idempotent）的自動合併設定，為了保持使用者體驗的一致性與支援的易用性，決定對 Copilot CLI hook 安裝進行自動化與格式修復。

## What Changes

1. 將 `tt setup --copilot` 升級為自動化冪等寫入，不再僅是列印指示說明。
2. 執行該命令時，`tt` 會自動在 `~/.copilot/hooks/tt.json` 寫入/合併最新的 Copilot hooks（包含 `userPromptSubmitted` 與 `agentStop`），並維持專案原有的 `mergeHooksFile` 冪等邏輯。

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `idempotent-hook-setup`: 新增 Copilot CLI hooks 自動設定與冪等合併需求。

## Impact

- Affected specs: `idempotent-hook-setup`
- Affected code:
  - New: (none)
  - Modified:
    - `internal/setup/setup.go`
    - `cmd/tt/setup_cmd.go`
    - `internal/setup/setup_test.go`
  - Removed: (none)

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-20-brainstorm-copilot-setup.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
