## Why

Copilot CLI session 的 token 計算與 model 歸因完全失效：DB 紀錄（`61232c8e`、`e0af7acb`）token 全 0 / NULL，且 model 被誤填為 `gemini-3.5-flash`。交叉比對真實 transcript 後確認三個根因：(1) `agentStop` hook 與 `session.shutdown` 寫入存在 ~400ms race，加上 `reconcile` WHERE 條件排除 NULL transcript_path 的 turn，導致永遠無法回補；(2) parser 用 json tag `mainModel` 但真實 event 欄位叫 `currentModel`，14 個 shutdown event 全 None；(3) `resolveModel` 用 Claude Code parser 解 Copilot transcript 失敗，fallback 到 Antigravity settings.json。

## What Changes

- **RC4 修復**：`copilotEvent.Data.MainModel` 為空時 fallback 到 `CurrentModel`；`vscode_copilot.go` shutdown parser 同修（兩處皆用 `json:"mainModel"` 但實際欄位是 `currentModel`）。
- **RC1 修復 — reconcile WHERE 放寬**：`reconcile.go` 的掃描條件新增 `OR tool='copilot-cli'`，讓 NULL transcript_path 的 Copilot turn 進入補算流程。
- **RC1 修復 — Copilot transcript path 自推**：`reconcileTurn` 對 `transcriptPath == ""` 的 turn 改用 `GetProvider(tool).ResolvePath(sessionID, "")` 推導 path（Copilot 靜態可推 `~/.copilot/session-state/<id>/events.jsonl`）；file 不存在則靜默 skip。
- **RC1 修復 — Copilot session 級 token 歸因**：因 Copilot `modelMetrics` 為 session 累計值（非 turn 增量），reconcile 對 `tool='copilot-cli'` session 改採歸因規則：清空所有 turn token → 最新累計值寫到最新 open turn（或最新 turn）→ 其餘 turn 補 `input_tokens=0`、`output_tokens=0`、`response_at`（`nextPromptAt - 1ms` 或自身 `prompt_at`）、`subagent_tokens_settled=1`。冪等、跨 shutdown 正確（report 加總 = session 實際 token）。
- **RC5 修復 — resolveModel provider 分流**：`resolveModel` 改用 `GetProvider(tool).ExtractWindow` 而非寫死 Claude Code parser；`repairSessions` 對 Copilot session 用 `ResolvePath` 推 path 餵給 `resolveModel`（`findExistingTranscriptPath` 對 Copilot 查無 transcript）。

## Non-Goals

- VS Code Copilot Chat token（本地無資料來源——transcript 無 `session.shutdown`、debug-logs 無 token、session.db 只有 todos）。
- VS Code session 被誤標 `copilot-cli`（根因 3，屬資料正確性範圍）。
- Turn 重複 4× 問題（根因 6）。
- Hook 重試 / 延遲機制。
- Schema migration。
- `record prompt` 存 transcript_path（reconcile 自推即可，不動 recorder）。

## Success Criteria

- **RC4 model fallback**：給定 `copilotEvent.Data.MainModel == ""` 且 `CurrentModel == "gpt-5"`，`ParseCopilotLog` 回傳的 `WindowResult.Model()` 為 `gpt-5`；`vscode_copilot.go` shutdown parser 同行為。
- **RC1 NULL path 進入 reconcile**：給定 `tool='copilot-cli'`、`transcript_path IS NULL`、`input_tokens IS NULL` 的 turn，且 `~/.copilot/session-state/<sessionID>/events.jsonl` 存在含 `session.shutdown` event，`MaybeReconcile` 後該 session 最新 open turn 的 `input_tokens` / `output_tokens` 等於 shutdown 累計值。
- **RC1 歸因規則**：給定 Copilot session 有 3 個 turn 且 shutdown `modelMetrics` 累計 `inputTokens=1000`，reconcile 後最新 turn `input_tokens=1000`、其餘兩 turn `input_tokens=0` 且 `subagent_tokens_settled=1`。
- **RC1 冪等**：對同一 Copilot session 連續執行兩次 `MaybeReconcile`，第二次所有 turn 的 token 值不變。
- **RC1 跨 shutdown**：給定 Copilot session 兩次 shutdown（第一次累計 1000、第二次累計 1500），reconcile 後 report 加總 = 1500（非 2500）。
- **RC5 model 分流**：給定 `tool='copilot-cli'` session 的 `turns.model` 為空或錯誤（如 `gemini-3.5-flash`），`repairSessions` 後 `turns.model` 對應 Copilot transcript `currentModel` 值。

## Capabilities

### New Capabilities

- `copilot-cli-transcript-parsing`: Parse Copilot CLI `events.jsonl` 格式，從 `session.shutdown` event 提取 model metrics 與 model 名稱（含 `mainModel` 空時 fallback `currentModel`）。

### Modified Capabilities

- `interrupt-reconcile`: reconcile WHERE 條件放寬接納 `tool='copilot-cli'` 的 NULL transcript_path turn；新增 Copilot session 級 token 歸因規則（累計值歸最新 open turn、其餘補 0）；`resolveModel` 改用 provider 分流而非寫死 Claude Code parser；`repairSessions` 對 Copilot session 用 `ResolvePath` 推 path。
- `vscode-copilot-session-parsing`: shutdown metric 提取新增 `currentModel` fallback（`mainModel` 為空時取 `currentModel`）。

## Impact

- Affected specs:
  - New: `openspec/specs/copilot-cli-transcript-parsing/spec.md`
  - Modified: `openspec/specs/interrupt-reconcile/spec.md`
  - Modified: `openspec/specs/vscode-copilot-session-parsing/spec.md`
- Affected code:
  - New: (none)
  - Modified:
    - `internal/transcript/copilot_transcript.go`
    - `internal/transcript/vscode_copilot.go`
    - `internal/reconcile/reconcile.go`
  - Removed: (none)

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-24-brainstorm-copilot-cli-token-fix.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
