## MODIFIED Requirements

### Requirement: Parse debug-logs JSONL format

系統 SHALL parse VS Code Copilot Chat debug log files (`debug-logs/{sessionId}/main.jsonl`) from workspaceStorage to extract actual token usage.

#### Scenario: Extract LLM request token counts

- **WHEN** a debug log JSONL file contains `llm_request` events
- **THEN** the system extracts inputTokens, outputTokens, cachedTokens, model from each event

#### Scenario: Extract session shutdown metrics

- **WHEN** a debug log JSONL file contains `session.shutdown` events
- **THEN** the system extracts per-model usage (inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens) and totalNanoAiu, and resolves the main model by using `data.mainModel` when present, falling back to `data.currentModel` when `mainModel` is empty

#### Scenario: mainModel empty falls back to currentModel

- **WHEN** a `session.shutdown` event has `data.mainModel == ""` and `data.currentModel == "gpt-5"`
- **THEN** the system uses `currentModel` (`gpt-5`) as the main model for subagent判定 and `WindowResult.Model()` returns `gpt-5`

#### Scenario: mainModel present takes precedence

- **WHEN** a `session.shutdown` event has `data.mainModel == "gpt-5"` and `data.currentModel == "claude-3.5"`
- **THEN** the system uses `mainModel` (`gpt-5`) and ignores `currentModel`

#### Scenario: Handle missing debug log

- **WHEN** the debug log directory does not exist for a session
- **THEN** the system returns nil and falls back to estimation
