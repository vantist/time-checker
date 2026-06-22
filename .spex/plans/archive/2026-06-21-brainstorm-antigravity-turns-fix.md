# antigravity-turns-fix

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

在 Google Antigravity 整合中，使用者回報資料庫中的 `turns` 與 `turn_model_usages` 紀錄有兩大異常：
1. **筆數異常過多**：一個 session 內只執行了幾次 user prompt，卻產生了數十個 `turns` 紀錄，且多數 `response_at` 為 `NULL`。
2. **數據皆為 0 與 unknown**：`turns` 與 `turn_model_usages` 的 Token 消耗皆為 0，且 Model 欄位顯示為 `"unknown"`。

經 codebase 追蹤與日誌排查，發現原因如下：
1. **觸發頻率不一致**：Antigravity 的 `PreInvocation` Hook 是在**每一次 LLM 調用（即 Agent 的每一個步驟）**時觸發，而非僅在 User Prompt 提交時。但 `Stop` Hook 僅在整個執行命令結束時觸發一次。這導致多次 prompt 被記錄為新 turn，但只有最後一筆 turn 能被 response 更新，其他都懸空（dangling）。
2. **Log 格式不匹配**：`internal/transcript/extract.go` 假設 Antigravity 的 `transcript.jsonl` 與 Claude Code 的格式相同（具有 `type == "assistant"` 與 `message.usage`），但真實的 Antigravity 日誌完全不包含 LLM Token 數據，其 `type` 為 `"PLANNER_RESPONSE"`。導致模型解析出 `"unknown"` 且 token 為 0。

## Decision

1. **Turn 紀錄去重**：在 `RecordPrompt` 中，針對 `tool == "antigravity"` 的請求，若該 session 已有尚未關閉的 turn（`response_at IS NULL`），則不重複插入新的 turn，將整個 Agent 的多個思考步驟合併為單一 Turn 紀錄。
2. **Model 正確取得與路徑修正**：比照 Copilot CLI 邏輯，將無法從 log 取得的 Token 數記錄為 0。但從 `~/.gemini/antigravity-cli/settings.json` 中讀取 `"model"` 鍵值並常態化（如 `"Gemini 3.5 Flash (Medium)"` -> `"gemini-3.5-flash"`）填入 model 欄位。更新 `ResolvePath` 優先尋找 `antigravity-cli` 目錄。

## Rationale

1. **對齊 Time Tracker 語意**：Time Tracker 的一個 turn 定義為一次 prompt-response 循環。將 internal LLM calls 隱藏在單個 turn 下，才能精確反應使用者的對話輪次。
2. **穩定可靠的 Model 讀取**：Antigravity 設定檔包含目前選用的模型。透過 `settings.json` 讀取並常態化，能保證報表中模型顯示正確（而非 "unknown"），且維持 0 token 運作，不會造成程式碼崩潰或估算高額錯誤費用。

## Approach

1. **去重邏輯**：
   在 `internal/recorder/recorder.go` 中，於 `INSERT` 前加入判斷：
   ```go
   if input.Tool == "antigravity" {
       var activeID int64
       err = conn.QueryRow("SELECT id FROM turns WHERE session_id=? AND response_at IS NULL LIMIT 1", stableID).Scan(&activeID)
       if err == nil && activeID > 0 {
           return nil // skip duplicate prompt record
       }
   }
   ```
2. **Model 提取邏輯**：
   在 `internal/transcript/antigravity_transcript.go` 中，實作 `getAntigravityModel()` 讀取 `~/.gemini/antigravity-cli/settings.json`（ fallback 到 `~/.gemini/antigravity/settings.json`，預設為 `gemini-3.5-flash`）。
   修改 `ParseAntigravityLog` 使其只返回包含此 model、0 token 的 `WindowResult`。
3. **路徑探測邏輯**：
   在 `internal/transcript/provider.go` 中，修改 `AntigravityProvider.ResolvePath`：
   優先判斷 `~/.gemini/antigravity-cli/brain/...` 是否存在，若存在則回傳，否則 fallback 到舊的 `~/.gemini/antigravity/brain/...`。

## Design Notes

- **相容性**：此變更不修改 DB schema，且維持現有 interface。
- **測試更新**：需同步更新 `internal/transcript/antigravity_transcript_test.go` 與 `internal/recorder/recorder_test.go`。

## Insights to Capture

- `design.md`: 說明多步驟工具（如 Antigravity）的 turn 合併去重設計與 settings.json 模型讀取策略。
- `specs/hook-integration/spec.md`: 補充說明 `PreInvocation` 去重規格。

## Open Questions

（無，設計已收斂）
