# copilot-cli-transcript-parsing Specification

## Purpose
TBD - created by archiving change copilot-cli-token-fix. Update Purpose after archive.
## Requirements
### Requirement: Parse Copilot CLI events.jsonl format

系統 SHALL parse Copilot CLI session 的 `events.jsonl` 檔案（位於 `~/.copilot/session-state/<sessionID>/events.jsonl`），從 `session.shutdown` event 提取 model metrics 與 model 名稱。

#### Scenario: 從 session.shutdown 提取 model metrics

- **WHEN** `events.jsonl` 包含 `session.shutdown` event，其 `data.modelMetrics` 為 map（key 為 model name，value 含 `usage.inputTokens`、`usage.outputTokens`、`usage.cacheReadTokens`、`usage.cacheWriteTokens`、`usage.reasoningTokens`）
- **THEN** 系統將每個 model 的 metrics 累加進 `WindowResult.Usages`，`outputTokens` 含 `reasoningTokens`，並標記 subagent（model name ≠ main model 時 `IsSubagent=true`）

#### Scenario: mainModel 為空時 fallback 到 currentModel

- **WHEN** `session.shutdown` event 的 `data.mainModel` 為空字串，且 `data.currentModel` 有值（例如 `gpt-5`）
- **THEN** 系統以 `currentModel` 作為 main model 判定 subagent：model name ≠ `currentModel` 時 `IsSubagent=true`；`WindowResult.Model()` 回傳 `currentModel`

#### Scenario: mainModel 有值時優先使用

- **WHEN** `session.shutdown` event 的 `data.mainModel` 有值（例如 `gpt-5`），且 `data.currentModel` 也有值（例如 `claude-3.5`）
- **THEN** 系統以 `mainModel` 作為 main model，`currentModel` 被忽略；`WindowResult.Model()` 回傳 `mainModel`

#### Scenario: 由 sessionID 靜態推導 transcript path

- **WHEN** 呼叫 `CopilotProvider.ResolvePath(sessionID, "")` 且 `stdinPath` 為空
- **THEN** 系統回傳 `~/.copilot/session-state/<sessionID>/events.jsonl`

#### Scenario: stdin 傳入 path 時優先使用

- **WHEN** 呼叫 `CopilotProvider.ResolvePath(sessionID, stdinPath)` 且 `stdinPath` 非空
- **THEN** 系統回傳 `stdinPath`，不推導

#### Scenario: 處理損毀 JSON 行

- **WHEN** `events.jsonl` 包含無效 JSON 行
- **THEN** 系統跳過該行繼續處理其餘行，不拋出錯誤

#### Scenario: 處理空行

- **WHEN** `events.jsonl` 包含空行或空白字元行
- **THEN** 系統跳過該行繼續處理其餘行

#### Scenario: 檔案不存在時回傳錯誤

- **WHEN** `events.jsonl` 路徑不存在
- **THEN** 系統回傳錯誤（`ExtractWindow` 與 `ParseCopilotLog` 皆然），呼叫端負責靜默 skip

