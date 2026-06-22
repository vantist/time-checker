## MODIFIED Requirements

### Requirement: 從 transcript 抽取 model 並補寫至 session

`RecordResponse` (或對應的 `record response` 鉤子命令) SHALL 根據所使用的 `tool` 分流解析對應的本地日誌，並提取 model 與精確的 token 數據存入資料庫：
1. **Claude Code** (`tool == "claude-code"`): 解析 `~/.claude/projects/**/*.jsonl`，從 assistant entry 抽取 `message.model`、`inputTokens` 等欄位。
2. **GitHub Copilot CLI** (`tool == "copilot-cli"`): 解析 `~/.copilot/session-state/<sessionId>/events.jsonl`，篩選 `"type":"session.shutdown"` 的 `modelMetrics`，提取該模型的 `inputTokens`、`outputTokens`、`cacheReadTokens`、`cacheWriteTokens` 等。
   - 子代理判定：比對模型的 `model` 與 `mainModel`，若不一致，則將其歸屬為子代理 (is_subagent = true) 的 Token 消耗。
3. **Google Antigravity** (`tool == "antigravity"`): 解析 `~/.gemini/antigravity/brain/<sessionId>/.system_generated/logs/transcript.jsonl`，統計主 Agent 的 model usage。其餘結構與 Claude Code 的 JSONL 相同，透過 `JSONLProvider` 共享機制實作主子代理與 reconcile 的 token 追蹤。
4. **OpenAI Codex** (`tool == "codex"`): 透過 `JSONLProvider` 解析由 `stdin` 傳遞之 `transcript_path` 的 JSONL 檔案。

若 `sessions.model` 為空，則以抽取出的 model 值補寫更新。

#### Scenario: model 從 transcript 寫入 session (Claude Code)

- **WHEN** Stop hook 呼叫 `tt record response`，`tool` 為 `claude-code` 且 `sessions.model` 為空
- **THEN** `sessions.model` MUST 被更新為 transcript 中的 model 值

#### Scenario: Copilot CLI 日誌解析 modelMetrics

- **WHEN** Stop hook 呼叫 `tt record response`，`tool` 為 `copilot-cli`，且 `sessionId` 為 `xyz`
- **THEN** `tt` MUST 解析 `~/.copilot/session-state/xyz/events.jsonl`，並正確提取 `session.shutdown` 事件中 `gpt-5.4` 模型的 input/output/cache token 消耗與 model 名稱

#### Scenario: Antigravity 日誌解析

- **WHEN** Stop hook 呼叫 `tt record response`，`tool` 為 `antigravity`，且 `sessionId` 為 `abc`
- **THEN** `tt` MUST 解析 `~/.gemini/antigravity/brain/abc/.system_generated/logs/transcript.jsonl`，統計主 Agent 的模型 input/output token 消耗

#### Scenario: Codex 日誌解析

- **WHEN** Stop hook 呼叫 `tt record response`，`tool` 為 `codex`，且 `sessionId` 為 `codex-1`，且 `transcriptPath` 欄位指向 `temp-codex.jsonl`
- **THEN** `tt` MUST 透過 `JSONLProvider` 解析 `temp-codex.jsonl`，並正確提取 `gpt-5.5-codex` 的模型與 token 消耗

#### Scenario: Copilot CLI 子代理判定

- **WHEN** 呼叫 CopilotProvider 的 `ExtractWindow` 且 `events.jsonl` 中 `session.shutdown` 事件包含了非 `mainModel` (例如 `gpt-5-mini`) 的 `modelMetrics`
- **THEN** 回傳的 `WindowResult` 中 MUST 包含該非 `mainModel` 的 `ModelUsage`，且標記 `IsSubagent = true`

#### Scenario: model 已存在時不覆蓋

- **WHEN** `sessions.model` 已有值（非空字串）
- **THEN** UPDATE 不執行，既有 model 值不變

#### Scenario: transcript 無 model 欄位

- **WHEN** transcript 的 assistant entry 無 `message.model` 欄位（空字串或欄位不存在）
- **THEN** `sessions.model` 保持原值，tokens 記錄照常完成
