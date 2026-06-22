## 1. Testing (Failing Tests)

- [x] 1.1 在 `internal/transcript/extract_test.go` 中新增單元測試 `TestExtractWindow_CorruptAndTrailingLines`，測試空行、空白行、損毀 JSON 的解析容錯以及超過 64KB 且小於 1MB 的超長行解析。執行測試並確認其失敗（或卡死）。

## 2. Core Implementation (實作與驗證)

- [x] 2.1 修改 `internal/transcript/extract.go` 中的 `loadTranscript` 實作，改用 `bufio.Scanner` 逐行讀取，並配置 1MB 讀取緩衝區，在遇到 unmarshal 失敗或空行時靜默跳過。
- [x] 2.2 執行 `go test ./...` 驗證新舊測試皆順利通過，確認解析卡死問題解決且沒有無窮迴圈。
