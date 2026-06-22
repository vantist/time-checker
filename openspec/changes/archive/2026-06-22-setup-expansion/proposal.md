## Why

目前 `tt setup` 僅支援 `Claude Code` 與 `GitHub Copilot CLI` 的 hook/指令設定。然而，底層與 log 解析實作已能支援 `Google Antigravity` 與 `OpenAI Codex`，這造成了設定面與底層實作能力的不一致，應予擴充。

## What Changes

- 擴充 `tt setup` 指令，新增 `--antigravity` 與 `--codex` 參數支援，以冪等（Idempotent）方式分別合併至其全域設定檔（`~/.gemini/config/hooks.json` 與 `~/.codex/hooks.json`）。
- 於 `cmd/tt/record.go` 中解析來自 stdin 的 Antigravity 欄位 `conversationId` 與 `transcriptPath`，並將其正確對應至 `SessionID` 與 `TranscriptPath`。
- 在 `internal/setup/setup.go` 中提取並實作一個通用的 `mergeHooksFile` Helper，供 Claude Code、Antigravity 與 Codex 的 hooks 設定共用，減少重複程式碼，且確保存檔時的 `0o600` 權限安全寫入。

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `hook-integration`: 擴充支援 Google Antigravity 與 OpenAI Codex 的 Hook 整合規格要求與 Payload 欄位規格。
- `idempotent-hook-setup`: 擴充支援以冪等方式寫入 Antigravity 及 Codex 整合設定檔。

## Impact

- Affected specs:
  - `openspec/specs/hook-integration/spec.md`
  - `openspec/specs/idempotent-hook-setup/spec.md`
- Affected code:
  - Modified:
    - `cmd/tt/setup_cmd.go`
    - `cmd/tt/record.go`
    - `internal/setup/setup.go`
    - `design.md`

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-20-brainstorm-setup-expansion.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
