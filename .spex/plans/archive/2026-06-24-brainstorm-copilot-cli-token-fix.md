# Brainstorm: Copilot CLI token 計算修復

**Date**: 2026-06-24
**Status**: ready for `/spex-propose`
**Scope**: 修 Copilot CLI session 的 token 計算與 model 歸因（不含 VS Code Copilot Chat token）

## 問題診斷

兩筆 DB 紀錄（`61232c8e`、`e0af7acb`）token 全 0 / NULL。交叉比對真實 transcript 後，根因如下（僅列入本次範圍者）：

### RC1 — agentStop race + reconcile 救不回
- `agentStop` hook 在 `05:15:32.000Z` 觸發 `tt record response`，但 `session.shutdown` event 在 `05:15:32.399Z` 才寫入 `events.jsonl`（晚 399ms）→ 讀檔時無 shutdown → token=0。
- `userPromptSubmitted` stdin 不含 `transcriptPath`（design.md 只記 agentStop 有）→ `record prompt` 存了 NULL `transcript_path`。
- `reconcile.go` 的 `WHERE transcript_path IS NOT NULL AND prompt_line_offset IS NOT NULL` 把這類 turn 排除 → 永遠無法回補。

### RC4 — model 欄位名不符
- parser 用 json tag `mainModel`，但真實 event 欄位叫 `currentModel`。驗證 14 個 shutdown event，`mainModel` 全 None。
- 影響：subagent 判定失效、`WindowResult.Model()` 回空。

### RC5 — resolveModel fallback 鏈對 Copilot 不適用
- `reconcile.go:491` 用 Claude Code parser 解 Copilot transcript 失敗 → 落 Antigravity settings.json → fallback `gemini-3.5-flash`。
- `e0af7acb` 的 model 就是被這樣誤填。

## 不在範圍

- VS Code Copilot Chat token（本地無資料來源——transcript 無 `session.shutdown`、debug-logs 無 token、session.db 只有 todos）
- VS Code session 被誤標 `copilot-cli`（根因 3，屬資料正確性範圍）
- turn 重複 4× 問題（根因 6）
- hook 重試 / 延遲
- schema migration
- `record prompt` 存 transcript_path（reconcile 自推即可）

## 關鍵設計決策

### 決策 1：靠 reconcile 回補，不跟 race 對賭
`session.shutdown` 比 `agentStop` hook 晚 ~400ms 寫入。不論 hook 當下怎麼讀都會漏。reconcile 在下次 hook 或 `tt report` 時跑，此時 shutdown 早已寫好。token 晚幾秒出現可接受。

### 決策 2：Copilot transcript path 由 reconcile 自推
`CopilotProvider.ResolvePath(sessionID, "")` = `~/.copilot/session-state/<id>/events.jsonl`（靜態可推）。不需 hook 傳、不需動 recorder。reconcile 對 `tool='copilot-cli'` 的 turn 自己 ResolvePath。

### 決策 3：session 級累計 token 歸到單一 turn，每次 reconcile 重置
Copilot `modelMetrics` 是 session 累計值，非 turn 增量。直接塞每個 open turn 會 N× 重複計算。

**歸因規則**（reconcile 對 copilot-cli session）：
1. 該 session 所有 turn 的 token 清空（刪 `turn_model_usages`、`turns` token 欄位 NULL）
2. 最新累計值寫到**最新 open turn**（或最新 turn）
3. 其他 turn 補 `input_tokens=0`、`output_tokens=0`、`response_at`（用 `nextPromptAt - 1ms` 或自己 `prompt_at`）、`subagent_tokens_settled=1`

冪等、跨 shutdown 正確（report 加總 = 該 turn 的累計值 = session 實際 token）。

### 決策 4：currentModel fallback
`copilotEvent.Data.MainModel` 為空時 fallback 到 `CurrentModel`。`vscode_copilot.go` 同修。

### 決策 5：resolveModel 用 provider 分流
`resolveModel` 改用 `GetProvider(tool).ExtractWindow` 而非寫死 Claude Code parser。`repairSessions` 對 Copilot session 用 `ResolvePath` 推 path（`findExistingTranscriptPath` 對 Copilot 查無 transcript）。

## 改動點

| 檔案 | 改動 | 根因 |
|---|---|---|
| `internal/transcript/copilot_transcript.go:25` | 加 `currentModel` json tag + `mainModel` fallback | RC4 |
| `internal/transcript/vscode_copilot.go:47` | 同上 | RC4 |
| `internal/reconcile/reconcile.go:96-98` | WHERE 放寬：`OR tool='copilot-cli'` | RC1 |
| `internal/reconcile/reconcile.go:126` `reconcileTurn` | `dt.transcriptPath == ""` → `provider.ResolvePath(sessionID, "")`；file 不存在則靜默 skip | RC1 |
| `internal/reconcile/reconcile.go` 新增 Copilot 歸因 | 清空所有 turn token → 累計值寫最新 open turn → 其餘補 0 + `response_at` + `settled=1` | RC1 |
| `internal/reconcile/reconcile.go:491` `resolveModel` | 改用 `GetProvider(tool).ExtractWindow` 分流 | RC5 |
| `internal/reconcile/reconcile.go` `repairSessions` | 對 Copilot session 用 `ResolvePath` 推 path 餵給 `resolveModel` | RC5 |

## 測試

- `copilot_transcript_test.go`: 加 `currentModel` fallback case（`mainModel` 空、`currentModel` 有值 → `Model()` 回傳 `currentModel`）
- `reconcile_test.go`: Copilot session 案例——NULL path turn + shutdown 在檔尾 → reconcile 後最新 turn 拿 token、其餘補 0
- reconcile 冪等：第二次 reconcile 不重複寫（token 值不變）
- 跨 shutdown：兩次 shutdown（累計值成長）→ report 加總 = 最新累計值

## Open Questions

- `resolveModel` 的 `path` 參數對 Copilot session 要從 `ResolvePath` 推，`repairSessions` 呼叫處需調整——實作時確認 `findExistingTranscriptPath` 對 Copilot 的行為（可能查無 → 需在 `repairSessions` 加 Copilot 分支直接 `ResolvePath`）。
