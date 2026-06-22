# model-cost-tracking Specification

## Purpose
TBD - created by archiving change enrich-report-and-tt-serve. Update Purpose after archive.
## Requirements
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

### Requirement: pricing normalize 去除 gateway 前綴

`pricing.Calculate` SHALL 在查詢 pricing table 前對 model ID 執行 normalize：
1. 去除最後一個 `/` 之前的所有字元（gateway 前綴如 `vertex_ai/`）。
2. 動態裁切日期後綴 `-\d{8}$` 與常見版本後綴：`-latest`、`-preview`、`-exp`、`-\d{3}`（如 `-001`、`-002`）。

#### Scenario: vertex_ai 前綴 model 正確計算 cost

- **WHEN** model 為 `vertex_ai/claude-sonnet-4-6`
- **THEN** `pricing.Calculate` MUST 以 `claude-sonnet-4-6` 查詢 pricing table，回傳非 nil cost

#### Scenario: 無前綴 model 維持正確

- **WHEN** model 為 `claude-sonnet-4-6`（無前綴）
- **THEN** normalize 後仍為 `claude-sonnet-4-6`，pricing 查詢結果不變

#### Scenario: 未知 model 回傳 nil

- **WHEN** normalize 後的 model 不在 pricing table 中
- **THEN** `pricing.Calculate` 回傳 nil（不影響現有行為）

#### Scenario: 日期後綴裁切正確

- **WHEN** model 為 `claude-sonnet-4-6-20260620`
- **THEN** normalize 後為 `claude-sonnet-4-6`，pricing 查詢結果正確

#### Scenario: 版本與預覽後綴裁切正確

- **WHEN** model 為 `claude-3-5-sonnet-latest` 或者是 `claude-3-5-sonnet-preview`
- **THEN** normalize 後為 `claude-3-5-sonnet`，pricing 查詢結果正確

#### Scenario: 數字版本號與實驗性後綴裁切正確

- **WHEN** model 為 `gemini-1.5-pro-002` 或者是 `gemini-2.5-flash-exp`
- **THEN** normalize 後分別為 `gemini-1.5-pro` 與 `gemini-2.5-flash`，pricing 查詢結果正確

### Requirement: pricing table 更新至最新定價

pricing table SHALL 包含以下 model 及其正確定價（USD / MTok），key 使用不含日期後綴與版本後綴的短 ID：

| Model key | Input | Output | Cache read (0.1× 或 OpenAI 50% 折扣) | Cache write 5m (1.25×) |
|-----------|-------|--------|-------------------|------------------------|
| `claude-fable-5` | $10.00 | $50.00 | $1.00 | $12.50 |
| `claude-opus-4-8` | $5.00 | $25.00 | $0.50 | $6.25 |
| `claude-opus-4-7` | $5.00 | $25.00 | $0.50 | $6.25 |
| `claude-opus-4-6` | $5.00 | $25.00 | $0.50 | $6.25 |
| `claude-opus-4-5` | $5.00 | $25.00 | $0.50 | $6.25 |
| `claude-sonnet-4-6` | $3.00 | $15.00 | $0.30 | $3.75 |
| `claude-sonnet-4-5` | $3.00 | $15.00 | $0.30 | $3.75 |
| `claude-haiku-4-5` | $1.00 | $5.00 | $0.10 | $1.25 |
| `claude-haiku-3-5` | $0.80 | $4.00 | $0.08 | $1.00 |
| `gpt-5.4` | $5.00 | $15.00 | $0.50 | $6.25 |
| `gpt-5-mini` | $0.15 | $0.60 | $0.015 | $0.1875 |
| `claude-3-opus` | $15.00 | $75.00 | $1.50 | $18.75 |
| `claude-3-sonnet` | $3.00 | $15.00 | $0.30 | $3.75 |
| `claude-3-haiku` | $0.25 | $1.25 | $0.025 | $0.3125 |
| `claude-3-5-sonnet` | $3.00 | $15.00 | $0.30 | $3.75 |
| `claude-3-5-haiku` | $0.80 | $4.00 | $0.08 | $1.00 |
| `gpt-4o` | $2.50 | $10.00 | $1.25 | $0.00 |
| `gpt-4o-mini` | $0.15 | $0.60 | $0.075 | $0.00 |
| `o1` | $15.00 | $60.00 | $7.50 | $0.00 |
| `o1-mini` | $3.00 | $12.00 | $1.50 | $0.00 |
| `o3-mini` | $1.10 | $4.40 | $0.55 | $0.00 |
| `gpt-5.3-codex` | $1.75 | $14.00 | $0.875 | $0.00 |
| `gpt-5.4-codex` | $2.50 | $15.00 | $1.25 | $0.00 |
| `gpt-5.5-codex` | $5.00 | $30.00 | $2.50 | $0.00 |
| `gpt-5.4-mini` | $0.75 | $3.00 | $0.375 | $0.00 |
| `gpt-5.5` | $5.00 | $30.00 | $2.50 | $0.00 |
| `mai-code-1-flash` | $0.75 | $4.50 | $0.075 | $0.00 |
| `raptor-mini` | $0.25 | $2.00 | $0.025 | $0.00 |
| `grok-code-fast-1` | $1.00 | $2.00 | $0.10 | $0.00 |
| `gemini-1.5-pro` | $1.25 | $5.00 | $0.125 | $0.00 |
| `gemini-1.5-flash` | $0.075 | $0.30 | $0.0075 | $0.00 |
| `gemini-2.5-pro` | $1.25 | $10.00 | $0.125 | $0.00 |
| `gemini-2.5-flash` | $0.30 | $2.50 | $0.03 | $0.00 |
| `gemini-2.5-flash-lite` | $0.10 | $0.40 | $0.01 | $0.00 |
| `gemini-3-flash` | $0.50 | $3.00 | $0.05 | $0.00 |
| `gemini-3.1-pro` | $2.00 | $12.00 | $0.20 | $0.00 |
| `gemini-3.1-flash-lite` | $0.25 | $1.50 | $0.025 | $0.00 |
| `gemini-3.5-flash` | $1.50 | $9.00 | $0.15 | $0.00 |

#### Scenario: haiku-4-5 cost 計算正確

- **WHEN** model `claude-haiku-4-5`，input 1,000,000 tokens，其餘 0
- **THEN** cost = $1.00（1 MTok × $1.00）

#### Scenario: opus-4-8 使用新定價

- **WHEN** model `claude-opus-4-8`，input 1,000,000 tokens，其餘 0
- **THEN** cost = $5.00（舊定價為 $15.00，新定價 $5.00）

#### Scenario: gpt-5.4 cost 計算正確

- **WHEN** model `gpt-5.4`，input 1,000,000 tokens，其餘 0
- **THEN** cost = $5.00（1 MTok × $5.00）

#### Scenario: gpt-5-mini cost 計算正確

- **WHEN** model `gpt-5-mini`，input 1,000,000 tokens，其餘 0
- **THEN** cost = $0.15（1 MTok × $0.15）

#### Scenario: gemini-3.5-flash cost 計算正確

- **WHEN** model `gemini-3.5-flash`，input 1,000,000 tokens，其餘 0
- **THEN** cost = $1.50（1 MTok × $1.50）

#### Scenario: claude-3-5-sonnet cost 計算正確且支援 cache write

- **WHEN** model `claude-3-5-sonnet`，input 1,000,000 tokens，cache write 1,000,000 tokens，其餘 0
- **THEN** cost = $3.00 + $3.75 = $6.75

