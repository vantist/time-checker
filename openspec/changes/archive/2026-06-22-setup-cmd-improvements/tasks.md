## 1. 智慧自動偵測 (Smart Auto-Detection)

- [x] 1.1 在 `internal/setup/setup_test.go` 中新增偵測函數的 TDD 測試案例（初期應失敗）
- [x] 1.2 在 `internal/setup/setup.go` 中實作偵測函數 `IsClaudeCodeActive`、`IsCopilotActive`、`IsAntigravityActive`、`IsCodexActive`
- [x] 1.3 執行測試並驗證偵測功能正確通過

## 2. 指令重構與多工具設定 (Command Refactoring & Multi-Tool Setup)

- [x] 2.1 在 `cmd/tt/setup_cmd_test.go` 中新增 TDD 測試案例，驗證傳入多個 flags 時能依序成功設定所有對應工具，以及未偵測到任何工具時顯示提示訊息的行為（初期應失敗）
- [x] 2.2 修改 `cmd/tt/setup_cmd.go` 中的 `RunE` 邏輯，重構以支援多工具並行設定、智慧偵測與提示訊息
- [x] 2.3 執行所有測試並驗證全數通過
