## 1. Recorder Turn 去重實作 (TDD)

- [x] 1.1 在 `internal/recorder/recorder_test.go` 撰寫測試，驗證當 `tool` 為 `"antigravity"` 且 session 含有尚未關閉的 active turn（`response_at` 為 NULL）時，調用 `RecordPrompt` 不會重複插入新 turn
- [x] 1.2 在 `internal/recorder/recorder.go` 中實作去重判斷：若 `input.Tool == "antigravity"` 且存在 active turn，跳過重複插入 turn 的行為
- [x] 1.3 執行測試 `go test ./internal/recorder/...` 並確認測試全部通過

## 2. Antigravity Model 提取與常態化實作 (TDD)

- [x] 2.1 在 `internal/transcript/antigravity_transcript_test.go` 中新增測試，驗證從 `settings.json` 讀取並常態化 model 名稱，並驗證當 transcript 內無 token 數據時，不報錯且回傳 token 消耗為 0
- [x] 2.2 在 `internal/transcript/antigravity_transcript.go` 中實作 `getAntigravityModel()` 函數，讀取並解析 `~/.gemini/antigravity-cli/settings.json`（若不存在則 fallback 至 `~/.gemini/antigravity/settings.json`）
- [x] 2.3 修改 `ParseAntigravityLog`，使用常態化的 model 名稱，並回傳 input/output token 消耗均為 0 的 `WindowResult`
- [x] 2.4 執行測試 `go test ./internal/transcript/...` 並確認測試全部通過

## 3. Provider 路徑探測邏輯實作

- [x] 3.1 在 `internal/transcript/provider.go` 中修改 `AntigravityProvider.ResolvePath`，優先尋找 `~/.gemini/antigravity-cli/brain/...` 路徑，若不存在則 fallback 至舊的 `~/.gemini/antigravity/brain/...`
- [x] 3.2 在 `internal/transcript/provider.go` 中覆寫 `AntigravityProvider.ExtractWindow` 與 `ExtractLastTurn` 直接呼叫 `ParseAntigravityLog`，確保單一 Turn 狀態可被完整提取。

## 4. Reconcile 機制對零 Token 工具支援實作

- [x] 4.1 在 `internal/reconcile/reconcile_test.go` 中新增測試 `TestReconcile_AntigravityZeroTokens`，驗證零 Token 的 Antigravity 能夠被順利 Reconcile。
- [x] 4.2 在 `internal/reconcile/reconcile.go` 中修改 `reconcileTurn` 邏輯，排除對 `antigravity` 的零 Token 限制，使其能夠正確完成 Reconcile 更新。
