## Why

在比對 `tt` 對不同 AI 工具（Claude Code, GitHub Copilot CLI, Google Antigravity, OpenAI Codex）的相容性時，發現以下問題：
1. **Copilot CLI `transcript_path` 與 `cwd` 丟失**：錄製 Copilot CLI 的 prompt 時，`readStdinJSON` 沒有正確轉換與儲存 `transcriptPath` 與 `cwd`。這導致 `turns.transcript_path` 為 NULL，Reconcile 機制完全失效。
2. **缺乏統一抽象**：日誌解析邏輯零散於 `cmd/tt/record.go` 與 `internal/reconcile/reconcile.go`，且 `reconcile` 直接寫死調用 Claude JSONL 解析，使得 Copilot CLI 無法被 Reconcile。
3. **Subagent 歸因不完整**：僅 Claude Code 支援子代理 Token 統計，Antigravity 和 Copilot 均被忽略，即使這兩個工具皆具備子代理功能。

## What Changes

1. **修正 Copilot CLI 資料擷取**：修復 `readStdinJSON`，確保 Copilot CLI 的 `transcriptPath` 與 `cwd` 能被正常擷取並記錄到資料庫。
2. **重構並抽象 Token 擷取層**：導入統一的 `LogProvider` 介面與 Registry 機制，將不同工具的路徑解析、最後回合提取及增量窗口提取行為完全抽象化。
3. **實作 JSONL 共享 Provider 基底**：提取現有 JSONL 逐行與 Subagent 遞迴解析邏輯至 `JSONLProvider`，使 Claude Code、Google Antigravity 與 OpenAI Codex 可共享相同的 Token 追蹤與 Subagent 歸因能力。
4. **實作 Copilot 自訂 Provider**：透過讀取 `session.shutdown` 事件並比對 `mainModel` 來偵測子代理的 Token 消耗。
5. **整合 Registry**：在 `reconcile` 與 `record` 機制中改為調用 `GetProvider(tool)`，統一所有工具的日誌解析與 Token 計算流程。

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `model-cost-tracking`: 補充各工具對 subagent 計算與 reconcile 的規範，並新增對 OpenAI Codex 模型的 Token 追蹤支援。

## Impact

- Affected specs:
  - `openspec/specs/model-cost-tracking/spec.md`
- Affected code:
  - New:
    - `internal/transcript/provider.go`
  - Modified:
    - `cmd/tt/record.go`
    - `internal/reconcile/reconcile.go`
    - `internal/transcript/extract.go`
  - Removed:
    - (none)

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-20-brainstorm-tool-log-provider.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
