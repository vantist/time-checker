## 1. 驗證 --resume 與 Session 復原

- [x] 1.1 在 `internal/db/session_test.go` 中新增或補充 `TestUpsertSession_Resume` 測試，驗證進程重新啟動但 conversation_id 相同時，成功更新 PID/Start 鍵並對齊 Session
- [x] 1.2 執行並驗證該測試通過

## 2. 驗證 Turn 隔離與去重機制

- [x] 2.1 在 `internal/recorder/recorder_test.go` 中新增或補充 `TestRecordPrompt_StableSession` 測試，驗證 `/clear` 之後能成功隔離為獨立的新 Turn，且在多步思考時能正常去重為單一 Turn
- [x] 2.2 執行並驗證該測試通過

## 3. 驗證 Reconcile 與 Clear Race 補救

- [x] 3.1 在 `internal/transcript/extract_test.go` 中新增或補充 `TestExtractLastTurn_ClearRace` 測試，驗證日誌路徑切換時的 Reconcile 限制與 Clear Race 時的 Fallback 讀取行為
- [x] 3.2 執行並驗證該測試通過

## 4. 系統整合測試與規格驗證

- [x] 4.1 執行專案完整單元測試 `go test ./...` 確保所有功能與歷史行為皆無衝突且全部通過
