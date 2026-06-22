## MODIFIED Requirements

### Requirement: 從 transcript 抽取 model 並補寫至 session

`RecordResponse` (或對應的 `record response` 鉤子命令) SHALL 根據所使用的 `tool` 分流解析對應的本地日誌（JSONL 格式），並提取 model 與精確的 token 數據存入資料庫：
1. **Claude Code** (`tool == "claude-code"`): 解析 `~/.claude/projects/**/*.jsonl`，從 assistant entry 抽取 `message.model`、`inputTokens` 等欄位。
2. **GitHub Copilot CLI** (`tool == "copilot-cli"`): 解析 `~/.copilot/session-state/<sessionId>/events.jsonl`，篩選 `"type":"session.shutdown"` 的 `modelMetrics`，提取該模型的 `inputTokens`、`outputTokens`、`cacheReadTokens`、`cacheWriteTokens` 等。
3. **Antigravity** (`tool == "antigravity"`): 優先解析 `~/.gemini/antigravity-cli/brain/<sessionId>/.system_generated/logs/transcript.jsonl`（若不存在則 fallback 至 `~/.gemini/antigravity/brain/<sessionId>/.system_generated/logs/transcript.jsonl`），以讀取 `settings.json` 來統計與補寫主 Agent 的 model 資訊與 token 消耗。

若 `sessions.model` 為空，則以抽取出的 model 值補寫更新。

#### Scenario: model 從 transcript 寫入 session (Claude Code)

- **WHEN** Stop hook 呼叫 `tt record response`，`tool` 為 `claude-code` 且 `sessions.model` 為空
- **THEN** `sessions.model` MUST 被更新為 transcript 中的 model 值

#### Scenario: Copilot CLI 日誌解析 modelMetrics

- **WHEN** Stop hook 呼叫 `tt record response`，`tool` 為 `copilot-cli`，且 `sessionId` 為 `xyz`
- **THEN** `tt` MUST 解析 `~/.copilot/session-state/xyz/events.jsonl`，並正確提取 `session.shutdown` 事件中 `gpt-5.4` 模型的 input/output/cache token 消耗與 model 名稱

#### Scenario: Antigravity 日誌解析與 settings.json 讀取

- **WHEN** Stop hook 呼叫 `tt record response`，`tool` 為 `antigravity`，且 `sessionId` 為 `abc`
- **THEN** `tt` MUST 探測並解析 `~/.gemini/antigravity-cli/brain/abc/.system_generated/logs/transcript.jsonl`（若不存在 fallback 至 `~/.gemini/antigravity/brain/abc/...`）
- **THEN** `tt` MUST 自 `~/.gemini/antigravity-cli/settings.json`（若不存在 fallback 至 `~/.gemini/antigravity/settings.json`）讀取 `model` 值（例如 `"Gemini 3.5 Flash (Medium)"`）並常態化為 `"gemini-3.5-flash"` 更新 turn/session 的 model 欄位
- **THEN** `tt` 將對應的 token 消耗記錄為 0，且不因日誌不包含 token 資料而報錯

#### Scenario: model 已存在時不覆蓋

- **WHEN** `sessions.model` 已有值（非空字串）
- **THEN** UPDATE 不執行，既有 model 值不變

#### Scenario: transcript 無 model 欄位

- **WHEN** transcript 的 assistant entry 無 `message.model` 欄位（空字串或欄位不存在）
- **THEN** `sessions.model` 保持原值，tokens 記錄照常完成
