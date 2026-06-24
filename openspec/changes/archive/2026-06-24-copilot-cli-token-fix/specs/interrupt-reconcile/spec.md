## MODIFIED Requirements

### Requirement: 補算懸空 turn 的 token 與結束時間

系統 SHALL 提供 `MaybeReconcile(conn *sql.DB)` 函式，掃描所有符合下列任一條件的 turn，並在 process 結束後從 transcript 重算 token（含 subagent）寫回 DB：

1. `response_at IS NULL`（Stop hook 未執行）
2. `input_tokens IS NULL`（token 未寫入）
3. `subagent_tokens_settled = 0`（subagent token 待重算）

且該 turn 具備 `transcript_path` 與 `prompt_line_offset`，**或** 該 turn 所屬 session 的 `tool = 'copilot-cli'`（此類 turn 的 `transcript_path` 與 `prompt_line_offset` 可能為 NULL，由 reconcile 透過 provider 自推 path）。

#### Scenario: 中間懸空 turn 補算成功

- **WHEN** DB 中存在 `response_at IS NULL` 的 turn，且該 turn 存在後繼 turn（`next_prompt_at` 不為 NULL）
- **THEN** 系統從 transcript 提取 `[prompt_line_offset, next_offset)` 的 token 窗口（`WindowResult`），將 `response_at` 設為 `next_prompt_at - 1ms`，並 UPDATE turn row（input_tokens、output_tokens、cache_read_tokens、cache_creation_tokens、cache_creation_5m_tokens、cache_creation_1h_tokens、model、estimated_cost_usd、response_at、subagent_tokens_settled=1）

#### Scenario: 最後一個懸空 turn（process 已死）補算成功

- **WHEN** DB 中存在 `response_at IS NULL` 的 turn，且該 turn 為 session 內最後一個（無後繼 turn），且對應 process 已不存活
- **THEN** 系統從 transcript 提取 `[prompt_line_offset, EOF)` 的 token 窗口，將 `response_at` 設為 transcript 檔案的 mtime，並 UPDATE turn row（含 `subagent_tokens_settled=1`）

#### Scenario: 進行中的 turn 不被誤算

- **WHEN** DB 中存在 `response_at IS NULL` 的 turn，且該 turn 為 session 內最後一個，且對應 process 仍存活（`process.IsAlive` 回傳 true），且該 turn 的 `prompt_at` 距今在 15 分鐘以內
- **THEN** 系統 skip 該 turn，不做任何 UPDATE

#### Scenario: 超時的進行中 turn 強制補算

- **WHEN** DB 中存在 `response_at IS NULL` 的 turn，且該 turn 為 session 內最後一個，且對應 process 仍存活，但該 turn 的 `prompt_at` 距今大於 15 分鐘
- **THEN** 系統不 skip 該 turn，強制進行 reconcile 將 `response_at` 更新為該 transcript 檔案的 mtime（若無法讀取或為 0 則 fallback 為目前時間減 1ms），並 UPDATE turn row

#### Scenario: Stop hook 已寫 response_at 但 subagent_tokens_settled=0 時重算 token

- **WHEN** `MaybeReconcile` 執行時，某 turn 的 `response_at` 已被 Stop hook 寫入（非 NULL），`input_tokens IS NOT NULL`，但 `subagent_tokens_settled = 0`，且 process 已不存活
- **THEN** reconcile 重新執行 `ExtractWindow`，覆蓋 token 欄位（包含正確的 subagent token），並將 `subagent_tokens_settled` 設為 1

#### Scenario: subagent_tokens_settled=1 的 turn 不被重算

- **WHEN** `MaybeReconcile` 執行時，某 turn 的 `response_at IS NOT NULL`、`input_tokens IS NOT NULL`、`subagent_tokens_settled = 1`
- **THEN** reconcile WHERE 條件不匹配該 turn，不做任何 UPDATE（no-op）

#### Scenario: Idempotency — 同一 turn 多次重算結果一致

- **WHEN** 相同 transcript 的同一 turn 被 `MaybeReconcile` 重算兩次
- **THEN** 第二次 UPDATE 產生相同結果，不累加或重複計算

#### Scenario: Copilot CLI turn（NULL transcript_path）進入 reconcile

- **WHEN** DB 中存在 `tool = 'copilot-cli'`、`transcript_path IS NULL`、`input_tokens IS NULL` 的 turn，且 `~/.copilot/session-state/<sessionID>/events.jsonl` 存在含 `session.shutdown` event
- **THEN** reconcile WHERE 條件（含 `OR tool='copilot-cli'`）匹配該 turn，系統透過 `CopilotProvider.ResolvePath(sessionID, "")` 推導 transcript path 並執行歸因流程

#### Scenario: Copilot CLI transcript 不存在時靜默 skip

- **WHEN** reconcile 推導出 Copilot transcript path 但 `os.Stat` 失敗（檔案不存在）
- **THEN** 系統 skip 該 turn，不拋出錯誤，不 UPDATE

#### Scenario: Copilot session 級累計 token 歸到最新 open turn

- **WHEN** reconcile 處理 `tool = 'copilot-cli'` session，該 session 有多個 turn 且 `session.shutdown` 的 `modelMetrics` 累計 `inputTokens=1000`、`outputTokens=500`
- **THEN** 系統先清空該 session 所有 turn 的 token 欄位與 `turn_model_usages`，將累計值（1000/500）寫到最新 open turn（`response_at IS NULL` 者；若無則最新 turn），該 turn `subagent_tokens_settled=1`

#### Scenario: Copilot session 其餘 turn 補 0 與 response_at

- **WHEN** reconcile 完成最新 open turn 的 token 歸因後，該 session 尚有其他 turn 未補值
- **THEN** 系統將這些 turn 的 `input_tokens=0`、`output_tokens=0`、`subagent_tokens_settled=1`，`response_at` 設為 `nextPromptAt - 1ms`（無後繼 turn 則用自身 `prompt_at`）

#### Scenario: Copilot session reconcile 冪等

- **WHEN** 對同一 `tool = 'copilot-cli'` session 連續執行兩次 `MaybeReconcile`，且 `session.shutdown` 累計值不變
- **THEN** 第二次 reconcile 所有 turn 的 token 值與第一次結果一致（重置後寫入同一累計值）

#### Scenario: Copilot session 跨 shutdown 累計正確

- **WHEN** Copilot session 有兩次 `session.shutdown`（第一次累計 `inputTokens=1000`、第二次累計 `inputTokens=1500`），reconcile 後執行 report 加總
- **THEN** report 的 session 總 `inputTokens=1500`（最新累計值，非 2500）

## ADDED Requirements

### Requirement: resolveModel 依工具 provider 分流

系統 SHALL 在 `resolveModel` 中依 session 的 `tool` 分流至對應 provider 的 `ExtractWindow`，而非寫死 Claude Code parser。

#### Scenario: Copilot session 用 CopilotProvider 解 model

- **WHEN** `resolveModel` 處理 `tool = 'copilot-cli'` 的 session，且該 session transcript 存在含 `session.shutdown` event
- **THEN** 系統呼叫 `GetProvider("copilot-cli").ExtractWindow` 提取 model，不 fallback 到 Antigravity `settings.json` 或 `gemini-3.5-flash`

#### Scenario: Claude Code session維持原行為

- **WHEN** `resolveModel` 處理 `tool = 'claude-code'` 的 session
- **THEN** 系統呼叫 `GetProvider("claude-code").ExtractWindow`，行為與重構前一致

### Requirement: repairSessions 推導 Copilot transcript path

系統 SHALL 在 `repairSessions` 中對 `tool = 'copilot-cli'` 的 session，當 `findExistingTranscriptPath` 查無 transcript 時，改用 `CopilotProvider.ResolvePath(sessionID, "")` 推導 path 餵給 `resolveModel`。

#### Scenario: findExistingTranscriptPath 查無時用 ResolvePath

- **WHEN** `repairSessions` 處理 `tool = 'copilot-cli'` session，且 `findExistingTranscriptPath` 回空
- **THEN** 系統呼叫 `CopilotProvider.ResolvePath(sessionID, "")` 推導 `~/.copilot/session-state/<sessionID>/events.jsonl` 並傳給 `resolveModel`

#### Scenario: findExistingTranscriptPath 找到時不覆蓋

- **WHEN** `repairSessions` 處理 `tool = 'copilot-cli'` session，且 `findExistingTranscriptPath` 回傳非空 path
- **THEN** 系統使用該 path，不呼叫 `ResolvePath` 覆蓋
