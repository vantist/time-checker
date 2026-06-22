# antigravity-session-fix

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

在 Google Antigravity 整合中，使用者回報了兩大問題：
1. **專案路徑未抓到**：在建立 session 時，`sessions.project` 欄位為空 `""`。這是因為 Antigravity 的 `PreInvocation` hook 的 stdin JSON payload 並不包含 `cwd` 欄位，導致 `resolvePromptInput` 解析出的 `project` 為空。
2. **其中一個 session 未抓到 model**：資料庫中有一筆 session（`62aaca0f-a987-490e-8239-152fe22cb68f`）的 `sessions.model` 欄位為空。這是因為該 session 對應的 `agy` 處理程序目前仍在背景運行中，並未觸發 `Stop` hook（未呼叫 `RecordResponse`），而後續背景 `reconcile` 雖然修補了 turns 且填入了 `turns.model`，卻沒有同步回填 `sessions.model` 欄位。

## Decision

1. **專案路徑 Fallback 至 `os.Getwd()`**：在 `resolvePromptInput` 中，若從 CLI 參數與 stdin JSON 均未取得 `project` 路徑，則 fallback 使用當前行程的 working directory `os.Getwd()`。
2. **建立 Prompt 時主動載入設定檔之 Model**：將 `internal/transcript/antigravity_transcript.go` 中的 `getAntigravityModel` 導出為 `GetAntigravityModel`。在 `resolvePromptInput` 中，若 `tool == "antigravity"` 且 `model == ""`，則呼叫該函數從 `settings.json` 中讀取預設模型寫入 session，避免初始 model 為空。
3. **Reconcile 成功時回填 `sessions.model`**：在 `internal/reconcile/reconcile.go` 中的 `reconcileTurn` 函數裡，若解析出的 turn model 不為空，則同步更新 `sessions.model`（僅在原本為空或為 NULL 時更新）。

## Rationale

1. **提升路徑追蹤的魯棒性**： hook 指令在啟動時，其 cwd 即為目前 AI 工具執行的專案目錄。因此 `os.Getwd()` 是一個十分安全且合理的 fallback。
2. **防止 model 顯示為空或 unknown**：即使 Stop hook 沒有成功觸發（例如進程在背景長駐），藉由在 Prompt 階段主動載入設定檔模型，並在 Reconcile 階段補回 model，能夠確保 session 的模型欄位隨時保持正確。

## Approach

### 1. `resolvePromptInput` 調整 (`cmd/tt/record.go`)
- 在解析完 flags 與 stdin 之後，若 `project == ""`，則呼叫 `os.Getwd()`。
- 若 `tool == "antigravity"` 且 `model == ""`，則導入並呼叫 `transcript.GetAntigravityModel(nil)` 來解析模型。

### 2. 導出 `GetAntigravityModel` (`internal/transcript/antigravity_transcript.go`)
- 將 `getAntigravityModel` 改名為 `GetAntigravityModel`。
- 修改 `ParseAntigravityLog` 以呼叫導出後的名稱。
- 更新單元測試 `internal/transcript/antigravity_transcript_test.go`。

### 3. Reconcile 回填邏輯 (`internal/reconcile/reconcile.go`)
- 在 `reconcileTurn` 的事務提交前，加入 SQL 更新：
  ```go
  if result.Model() != "" {
      _, err = tx.Exec(`UPDATE sessions SET model=? WHERE id=? AND (model='' OR model IS NULL)`, result.Model(), dt.sessionID)
      if err != nil {
          return err
      }
  }
  ```

## Design Notes

- **相容性**：此變更不修改 DB schema，且維持現有 interface。
- **測試**：需同步更新 `internal/transcript/antigravity_transcript_test.go` 確保無誤。

## Insights to Capture

- `design.md`: 補充說明 Antigravity 整合中專案路徑的 fallback 機制與 sessions.model 的 reconcile 回填策略。
- `specs/hook-integration/spec.md`: 補充說明 project path 與 model name 的解析規範。

## Open Questions

（無，設計已收斂）
