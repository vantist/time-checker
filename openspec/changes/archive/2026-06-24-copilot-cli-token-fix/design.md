## Context

`tt` 透過 hook 自動記錄 AI 工具的 turn token 與 model。Claude Code 與 Antigravity 運作正常，但 Copilot CLI 因為以下三個根因完全失效：

- **RC1（race + WHERE 排除）**：`agentStop` hook 在 `session.shutdown` event 寫入 `events.jsonl` 前 ~400ms 觸發 `tt record response`，讀檔時無 shutdown → token=0。`userPromptSubmitted` stdin 不含 `transcriptPath` → `record prompt` 存 NULL `transcript_path`。`reconcile.go` 的 `WHERE transcript_path IS NOT NULL AND prompt_line_offset IS NOT NULL` 把這類 turn 永久排除。
- **RC4（model 欄位名不符）**：`copilot_transcript.go:25` 與 `vscode_copilot.go:47` 用 json tag `mainModel`，但真實 event 欄位叫 `currentModel`。驗證 14 個 shutdown event，`mainModel` 全 None → `WindowResult.Model()` 回空、subagent 判定失效。
- **RC5（resolveModel fallback 鏈不適用）**：`reconcile.go` 的 `resolveModel` 用 Claude Code parser 解 Copilot transcript 失敗 → 落 Antigravity `settings.json` → fallback `gemini-3.5-flash`。`e0af7acb` 的 model 就是被這樣誤填。

現有 `interrupt-reconcile` 規格已涵蓋 Claude Code / Antigravity 的 reconcile 行為，但未涵蓋 Copilot CLI 的兩個特殊性：(1) session 級累計 token（非 turn 增量），(2) transcript path 可由 sessionID 靜態推導。

## Goals / Non-Goals

**Goals:**

- 讓 Copilot CLI session 的 token 在 reconcile 後正確寫入（即使 hook 當下因 race 漏抓）。
- 讓 Copilot CLI session 的 model 正確歸因（不再被誤填 `gemini-3.5-flash`）。
- 修正 `copilot_transcript.go` 與 `vscode_copilot.go` 的 `mainModel`/`currentModel` 欄位對應。
- 維持 reconcile 冪等性與跨 shutdown 正確性。
- 不動 recorder、不動 schema、不動 hook stdin 契約。

**Non-Goals:**

- VS Code Copilot Chat token（無資料來源）。
- VS Code session 被誤標 `copilot-cli`（根因 3，資料正確性範圍）。
- Turn 重複 4× 問題（根因 6）。
- Hook 重試 / 延遲機制。
- Schema migration。
- 讓 `record prompt` 存 transcript_path（reconcile 自推即可）。

## Decisions

### 決策 1：靠 reconcile 回補，不跟 race 對賭

`session.shutdown` 比 `agentStop` hook 晚 ~400ms 寫入。不論 hook 當下怎麼讀都會漏。reconcile 在下次 hook 或 `tt report` 時跑，此時 shutdown 早已寫好。token 晚幾秒出現可接受。

**替代方案考慮**：在 `tt record response` 加 500ms sleep 後重讀。否決——增加 hook 延遲、治標不治本（race window 仍存在）、且 hook 失敗靜默的設計原則不該被破壞。

### 決策 2：Copilot transcript path 由 reconcile 自推

`CopilotProvider.ResolvePath(sessionID, "")` = `~/.copilot/session-state/<id>/events.jsonl`（靜態可推）。reconcile 對 `tool='copilot-cli'` 且 `transcriptPath == ""` 的 turn 自己呼叫 `ResolvePath`，不需 hook 傳、不需動 recorder。

**替代方案考慮**：在 `userPromptSubmitted` hook stdin 加 `transcriptPath`。否決——需改 hook 契約與 recorder，影響面大於收益。

### 決策 3：session 級累計 token 歸到單一 turn，每次 reconcile 重置

Copilot `modelMetrics` 是 session 累計值，非 turn 增量。直接塞每個 open turn 會 N× 重複計算。

**歸因規則**（reconcile 對 `tool='copilot-cli'` session）：
1. 該 session 所有 turn 的 token 清空（刪 `turn_model_usages`、`turns` token 欄位 NULL）。
2. 最新累計值寫到**最新 open turn**（`response_at IS NULL` 者；若無則最新 turn）。
3. 其他 turn 補 `input_tokens=0`、`output_tokens=0`、`response_at`（用 `nextPromptAt - 1ms` 或自己 `prompt_at`）、`subagent_tokens_settled=1`。

冪等、跨 shutdown 正確（report 加總 = 該 turn 的累計值 = session 實際 token）。

**替代方案考慮**：差分累計值（本次 shutdown - 前次 shutdown）歸到當下 turn。否決——需追蹤前次 shutdown 值的 state、跨 process 重啟後 state 遺失、複雜度高於收益。

### 決策 4：currentModel fallback

`copilotEvent.Data.MainModel` 為空時 fallback 到 `CurrentModel`。`vscode_copilot.go` shutdown parser 同修（兩處欄位名一致）。

### 決策 5：resolveModel 用 provider 分流

`resolveModel` 改用 `GetProvider(tool).ExtractWindow` 而非寫死 Claude Code parser。`repairSessions` 對 Copilot session 用 `ResolvePath` 推 path（`findExistingTranscriptPath` 對 Copilot 查無 transcript）。

**替代方案考慮**：在 `resolveModel` 加 `if tool == "copilot-cli"` 分支。否決——硬編碼工具別違反 provider 抽象，未來新增工具又要改此函式。

## Risks / Trade-offs

- **[風險] Copilot session 有多個 open turn 時，token 全歸最新 turn 可能使個別 turn 報表失真** → 可接受。Copilot `modelMetrics` 無法拆 turn，這是資料來源限制。report 加總仍正確，符合「session 級成本正確」的核心目標。
- **[風險] reconcile 重置 turn token 會覆蓋先前正確值** → 冪等設計保證：重置後再寫入同一累計值，第二次 reconcile 結果不變。對於非 Copilot session，歸因規則不觸發（WHERE 條件限定 `tool='copilot-cli'`）。
- **[風險] `ResolvePath` 對不存在 sessionID 回傳不存在路徑** → `reconcileTurn` 偵測 `os.Stat` 失敗時靜默 skip（與現有 transcript 不存在行為一致）。
- **[權衡] currentModel fallback 改變 parser 既有行為** → 既有 `mainModel` 有值時優先，無值時才 fallback，向後相容。
- **[風險] `repairSessions` 對 Copilot 推 path 可能與 `findExistingTranscriptPath` 行為衝突** → `findExistingTranscriptPath` 對 Copilot 查無 transcript（回空），`repairSessions` 新分支只在 path 為空時用 `ResolvePath` 推導，不會覆蓋已找到的真實 path。

## Open Questions

- `resolveModel` 的 `path` 參數對 Copilot session 要從 `ResolvePath` 推，`repairSessions` 呼叫處需調整——實作時確認 `findExistingTranscriptPath` 對 Copilot 的行為（可能查無 → 需在 `repairSessions` 加 Copilot 分支直接 `ResolvePath`）。
