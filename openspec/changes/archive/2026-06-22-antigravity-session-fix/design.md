## Context

當前 Google Antigravity 整合中，主要存在兩個問題：
1. **專案路徑解析失敗**：在建立 session 時，因為 `PreInvocation` hook 的 stdin JSON payload 不包含 `cwd` 欄位，導致 `resolvePromptInput` 解析出來的專案路徑為空。
2. **session.model 遺失**：若進程在背景長駐運行，並未觸發 `Stop` hook（未呼叫 `RecordResponse`），則 `sessions.model` 會為空。即使後續 reconcile 修補了 turns 且填入了 `turns.model`，也沒有同步回填 `sessions.model` 欄位。

## Goals / Non-Goals

**Goals:**
- 提供 `sessions.project` 的安全 Fallback 機制，自動解析為當前工作目錄。
- 提供 `sessions.model` 的自動載入與 Reconcile 回填機制，確保 session model 資料完整。
- 維持現有的資料庫結構，不修改 schema。

**Non-Goals:**
- 不修改 PreInvocation hook 的 payload 格式。
- 不修改/不重構與本次問題無關的 session 生命週期管理邏輯。

## Decisions

### 1. 專案路徑使用 `os.Getwd()` 作為 Fallback
- **Rationale**: CLI hook 執行時，當前進程的工作目錄即為使用者的專案目錄。因此當 stdin JSON 與 CLI flags 皆未提供專案路徑時，使用 `os.Getwd()` 是安全且合理的 fallback。
- **Alternatives**: 不做 fallback（會導致專案欄位持續為空，失去追蹤專案時間的效益）。

### 2. 導出並呼叫 `GetAntigravityModel`
- **Rationale**: 導出 `internal/transcript/antigravity_transcript.go` 中的 `getAntigravityModel` 為 `GetAntigravityModel`。在建立 Prompt 時，若發現為 `antigravity` 工具且 model 為空，則主動讀取 `settings.json` 中配置的預設 model。
- **Alternatives**: 僅依靠 reconcile 回填（這會導致 session model 在 reconcile 執行前保持為空，無法即時在報表中呈現）。

### 3. Reconcile 階段回填 `sessions.model`
- **Rationale**: 在 `internal/reconcile/reconcile.go` 中的 `reconcileTurn` 函數內，若解析出的 turn model 不為空，則同步更新 `sessions` 資料表中的 `model`（僅在原本為空或為 NULL 時更新）。此處有完整的 DB transaction 及對應的 session/turn 資訊，在此更新成本最低且最安全。
- **Alternatives**: 不回填（會導致部分背景運行的 session 即使有 turn 記錄也無法呈現 session model）。

## Risks / Trade-offs

- **[Risk]** `os.Getwd()` 因權限或其他原因返回錯誤。
  - **Mitigation** 若 `os.Getwd()` 發生 error，則記錄該 error 並使 project 為空，保證 hook 靜默且不阻擋主要流程。
- **[Risk]** 在 reconcile 中對 `sessions` 表進行 `UPDATE` 可能造成額外的寫入開銷。
  - **Mitigation** 該 SQL 僅在 `model = '' OR model IS NULL` 時才會被執行，且 reconcile 屬於背景非同步排程，不會影響 CLI 前台回應速度。
