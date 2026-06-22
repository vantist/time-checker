# tool-log-provider

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

在比對 `tt` (AI Tool Time Tracker) 對不同 AI 工具（Claude Code, GitHub Copilot CLI, Google Antigravity, OpenAI Codex）的相容性時，發現以下問題與功能不對等：
1. **Copilot CLI `transcript_path` 與 `cwd` 丟失**：在錄製 Copilot CLI 的 prompt 時，`readStdinJSON` 沒有正確轉換與儲存 `transcriptPath` 與 `cwd`。這導致 `turns.transcript_path` 為 NULL，Reconcile 機制完全失效。
2. **缺乏統一抽象**：日誌解析邏輯零散於 `cmd/tt/record.go` 與 `internal/reconcile/reconcile.go`，且 `reconcile` 直接寫死調用 Claude JSONL 解析，使得 Copilot CLI 無法被 Reconcile。
3. **Subagent 歸因不完整**：僅 Claude Code 支援子代理 Token 統計，Antigravity 和 Copilot 均被忽略，即使這兩個工具皆具備子代理功能。

## Decision

設計並重構 Token 擷取層，導入統一的 `LogProvider` 介面與 Registry 機制，將不同工具的路徑解析、最後回合提取及增量窗口提取行為完全抽象化。

## Rationale

1. **職責單一與高內聚**：將工具特定的日誌檔位置、格式解析與偏移量（Offset）對齊封裝在個別的 Provider 中，調用方（如 Reconcile 與 Recorder）只需調用通用介面。
2. **程式碼重用**：設計通用的 `JSONLProvider` 作為 Claude Code、Antigravity 與 Codex 的共享基底，讓後兩者立即可享有 Subagent 追蹤與 Turn-level Reconcile 能力。
3. **消除工具限制**：利用 `mainModel` 比對機制，解決 Copilot CLI 缺乏 turn-level subagent 標記的限制，實現跨模型子代理偵測。

## Approach

1. **LogProvider 介面**：
   定義 `ResolvePath`、`ExtractWindow`、`ExtractLastTurn` 與 `SupportsSubagents` 方法。
2. **JSONLProvider 共享基底**：
   封裝現有 `extract.go` 的 JSONL 逐行與 Subagent 遞迴解析邏輯。
3. **CopilotProvider 自訂解析**：
   讀取 `~/.copilot/session-state/.../events.jsonl` 中的 `session.shutdown` 事件，並依據是否與 `mainModel` 一致來區分並標記子代理的 Model 消耗。
4. **CodexProvider 共享實作**：
   直接嵌入 `JSONLProvider` 基底，並將 `ResolvePath` 指向 stdin 傳入的 `transcript_path`。因為 Codex 與 Claude Code 使用相同的 JSONL 結構，這能讓 Codex 立刻獲得同等規格的 Token 追蹤與 Subagent 支援。
5. **註冊中心與分流**：
   建立 `GetProvider(tool)` 自動返回對應適配器，確保調用端 `reconcile` 與 `record` 機制 100% 統一。

## Design Notes

### 介面設計 (`internal/transcript/provider.go`)
```go
type LogProvider interface {
    ResolvePath(sessionID string, stdinPath string) string
    ExtractWindow(path string, fromOffset int, toOffset int) (WindowResult, error)
    ExtractLastTurn(path string) (WindowResult, error)
    SupportsSubagents() bool
}
```

### 各工具 Provider 繼承與定位關係
1. **ClaudeProvider**：嵌入 `JSONLProvider`，`ResolvePath` 回傳 `stdinPath`。
2. **AntigravityProvider**：嵌入 `JSONLProvider`，`ResolvePath` 對齊 `~/.gemini/antigravity/...`。
3. **CodexProvider**：嵌入 `JSONLProvider`，`ResolvePath` 回傳 `stdinPath`。
4. **CopilotProvider**：獨立實作 `LogProvider`，自訂對齊 `~/.copilot/session-state/.../events.jsonl` 並包含跨模型 Subagent 辨識。

### 系統資料流對齊
1. **Prompt 錄製**：Copilot CLI 傳入的 `transcriptPath` 與 `cwd` 須正常 normalise 並寫入資料庫，以確保 `turns.transcript_path` 有值。
2. **Stop 錄製**：統一調用 `provider.ExtractLastTurn(path)`。
3. **Reconcile 補齊**：`reconcileTurn` 將調用 `provider.ExtractWindow(path, offset, to)`，從而支援 Copilot CLI 與其他工具的 Reconcile。

## Insights to Capture

- `design.md`: 補充 `LogProvider` 與 `JSONLProvider` 的架構設計細節。
- `specs/model-cost-tracking/spec.md`: 補充各工具對 subagent 計算與 reconcile 的規範。

## Open Questions

（無，設計已收斂）
