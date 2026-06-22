## Context

在支援多種 AI 工具（Claude Code, GitHub Copilot CLI, Google Antigravity, OpenAI Codex）的情況下，當前面臨了以下架構性的挑戰：
1. 日誌解析邏輯分散於 `cmd/tt/record.go` 與 `internal/reconcile/reconcile.go`，且 `reconcile` 模組寫死了呼叫 Claude JSONL 解析的邏輯。
2. Copilot CLI 資料因在錄製 prompt 時丟失了 `transcriptPath` 與 `cwd`，導致資料庫中的 `turns.transcript_path` 為 NULL，造成無法執行 Reconcile 的 Bug。
3. 僅 Claude Code 支援 subagent Token 統計，其他工具如 Antigravity 與 Copilot 雖支援子代理卻無法被 `tt` 追蹤與歸因。

## Goals / Non-Goals

**Goals:**
- 提供統一的 `LogProvider` 介面，支援多種 AI 工具的日誌路徑解析與 Token 統計。
- 實作共享的 `JSONLProvider` 用以處理 Claude, Antigravity 及 Codex 相同的 JSONL 日誌。
- 實作專門的 `CopilotProvider` 解析 Copilot 專屬的 `events.jsonl` 日誌，支援 subagent 計算與 reconcile。
- 修正 Copilot CLI prompt 錄製中丟失的 `transcript_path` 與 `cwd` 問題。
- 使用 `LogProvider` 與 Registry 機制重構 `reconcile` 和 `record` 邏輯。

**Non-Goals:**
- 不改變資料庫的 Turn / Session Schema 結構。
- 本次不支援非本機日誌或未列出 AI 工具的對齊解析。

## Decisions

1. **介面定義 (`internal/transcript/provider.go`)**
   定義 `LogProvider` 介面以統一解析行為：
   ```go
   type LogProvider interface {
       ResolvePath(sessionID string, stdinPath string) string
       ExtractWindow(path string, fromOffset int, toOffset int) (WindowResult, error)
       ExtractLastTurn(path string) (WindowResult, error)
       SupportsSubagents() bool
   }
   ```
2. **Registry 機制**
   於 `internal/transcript` 提供全域 Registry：
   ```go
   var providers = make(map[string]LogProvider)
   func Register(tool string, p LogProvider)
   func GetProvider(tool string) (LogProvider, bool)
   ```
   預設註冊：
   - `"claude-code"` -> `ClaudeProvider` (基於 `JSONLProvider`)
   - `"antigravity"` -> `AntigravityProvider` (基於 `JSONLProvider`)
   - `"codex"` -> `CodexProvider` (基於 `JSONLProvider`)
   - `"copilot-cli"` -> `CopilotProvider` (獨立實作)

3. **JSONL 共享實作 (`JSONLProvider`)**
   將原 `internal/transcript/extract.go` 內部的 JSONL 逐行讀取、最後回合提取以及 subagent 遞迴解析邏輯重構並封裝至 `JSONLProvider` 結構中。其他基於 JSONL 的工具（Claude Code, Antigravity, Codex）透過結構體嵌入 (Struct Embedding) 共享其實作。

4. **Copilot CLI 子代理判定與 Reconcile 整合**
   - 解決 `transcript_path` 丟失問題：修改 `cmd/tt/record.go` 中的 Stdin JSON 解析邏輯（如 `readStdinJSON`），確保傳入的 `transcriptPath` 與 `cwd` 正確映射到 Turn 結構中。
   - `CopilotProvider.ResolvePath` 將 `sessionID` 轉為絕對路徑 `~/.copilot/session-state/<sessionID>/events.jsonl`。
   - 子代理判定：由於 Copilot 事件不包含子代理的 tool use 標記，透過掃描 events.jsonl 中的 `modelMetrics`，如果其 model 與 mainModel 不一致，則歸類為子代理的 token。

## Risks / Trade-offs

- **[Risk]** 各工具的日誌檔路徑在不同系統可能有所不同。
  - **Mitigation**: 在各 Provider 的 `ResolvePath` 中優先使用環境變數或 stdin 傳遞的路徑，並確保對齊平台預設路徑。
- **[Risk]** Copilot 事件可能在 `reconcile` 執行時還未寫入完整。
  - **Mitigation**: 確保 `ExtractWindow` 回傳適當的錯誤，由調用方處理重試或靜默完成。
