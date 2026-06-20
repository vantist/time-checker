## Context

在 Google Antigravity 整合中，使用者回報資料庫中的 `turns` 與 `turn_model_usages` 紀錄有兩大異常：
1. **筆數異常過多**：一個 session 內只執行了幾次 user prompt，卻產生了數十個 `turns` 紀錄，且多數 `response_at` 為 `NULL`。這是因為 Antigravity 的 `PreInvocation` Hook 是在每一次 LLM 調用（即 Agent 的每一個步驟）時觸發，而非僅在 User Prompt 提交時。但 `Stop` Hook 僅在整個執行命令結束時觸發一次。這導致多次 prompt 被記錄為新 turn，但只有最後一筆 turn 能被 response 更新，其他都懸空（dangling）。
2. **數據皆為 0 與 unknown**：`turns` 與 `turn_model_usages` 的 Token 消耗皆為 0，且 Model 欄位顯示為 `"unknown"`。這是因為 Antigravity 的 `transcript.jsonl` 不包含 LLM Token 數據，其 `type` 為 `"PLANNER_RESPONSE"`，而 `internal/transcript/extract.go` 假設其與 Claude Code 格式相同（具有 `type == "assistant"` 與 `message.usage`），因而解析出 `"unknown"` 且 token 為 0。

## Goals / Non-Goals

**Goals:**
- 去除多餘的 `turns` 紀錄，使 Antigravity 每個對話 session 的多步驟思考合併為單一 Turn 紀錄。
- 正確自設定檔中讀取並常態化 Antigravity 的 `model` 欄位（例如 `"Gemini 3.5 Flash (Medium)"` -> `"gemini-3.5-flash"`），避免顯示為 `"unknown"`。
- 更新路徑探測邏輯，優先支援包含 `antigravity-cli` 的 brain path。
- 使用 TDD 開發，撰寫對應的單元測試並確保其通過。

**Non-Goals:**
- 不修改 Database Schema。
- 不估計/不抓取 Antigravity 的 Token 數（仍記為 0）。

## Decisions

### 1. Turn 紀錄去重

在 `internal/recorder/recorder.go` 中，於 `INSERT` 前加入判斷：
若 `input.Tool == "antigravity"`，查詢 `turns` 中是否有同一個 `session_id` 且 `response_at IS NULL` 的 active turn。如果存在，則代表該步驟屬於同一個 turn 的後續步驟（或者屬於懸空的 turn，但由於 `Stop` 尚未被調用，所有多個步驟應該合併），此時直接回傳 `nil`（跳過重複插入新的 turn 紀錄）。

*Alternatives Considered:*
- **在 Hook 本身進行防重**：例如由 cli 發送時帶入特別標記。但這樣會增加 Hook 程式複雜度，且如果 Hook 出錯容易影響 tool 運作。在 `recorder`（DB 寫入層）進行去重更為穩定。

### 2. Model 正確取得與常態化

在 `internal/transcript/antigravity_transcript.go` 中實作 `getAntigravityModel()`：
1. 尋找 `~/.gemini/antigravity-cli/settings.json`（若不存在，則 fallback 至 `~/.gemini/antigravity/settings.json`）。
2. 讀取並解析 JSON，取得 `"model"` 欄位的值。
3. 進行 Model Name 常態化（對齊 pricing table 中的 key）。例如若值為 `"Gemini 3.5 Flash (Medium)"` 則常態化為 `"gemini-3.5-flash"`。預設值為 `"gemini-3.5-flash"`。

*Alternatives Considered:*
- **從日誌中猜測**：但 Antigravity 的日誌只包含 `PLANNER_RESPONSE` 等，不包含 model parameter 等資訊。讀取 `settings.json` 是最穩定且準確的方式。

### 3. 路徑探測邏輯

在 `internal/transcript/provider.go` 中，修改 `AntigravityProvider.ResolvePath`：
優先判斷 `~/.gemini/antigravity-cli/brain/...` 是否存在，若存在則回傳此路徑，否則 fallback 至舊的 `~/.gemini/antigravity/brain/...`。

在 `AntigravityProvider` 中覆寫 `ExtractWindow` 與 `ExtractLastTurn` 直接使用 `ParseAntigravityLog(path)` 來讀取完整的 settings.json 與統計零 Token 消耗。

### 4. Reconcile 對零 Token 消耗工具的支援

在 `internal/reconcile/reconcile.go` 的 `reconcileTurn` 中，將原先 token 皆為 0 便直接跳過的條件調整為 `result.InputTokens() == 0 && result.OutputTokens() == 0 && tool != "antigravity"`。這允許 reconcile 可以為 Antigravity 回寫正確的 model 與時間資訊。

## Risks / Trade-offs

- **[Risk]** `settings.json` 讀取失敗或格式不符 → **[Mitigation]** 當讀取失敗時 fallback 到預設模型 `"gemini-3.5-flash"`，且寫入 debug log，不影響程式運作。
- **[Risk]** Session 被中斷導致 dangling turn 長期存在於資料庫 → **[Mitigation]** 當有新的 turn 寫入時，如果 `response_at` 為 NULL，合併僅限於 `tool == "antigravity"` 且同一個 `session_id`。如果 2 個不同的 session 或者是新的執行，會以新的 session 建立（session-management 本身有機制隔離不同 session 的執行）。同時利用 reconcile 自動在背景關閉並回寫這些 dangling turns。
- **[Risk]** `/clear` 與 `resume` 造成 Turn 的越界或 Token 重設 → **[Mitigation]** `db.UpsertSession` 有內建的 resume 進程鍵更新機制；`reconcileTurn` 在跨日誌檔邊界時會將 `toOffset` 設為 `-1` 讀取到檔案末端；且 `ExtractLastTurn` 設有 `ClearRace` fallback 機制保護舊資料。
