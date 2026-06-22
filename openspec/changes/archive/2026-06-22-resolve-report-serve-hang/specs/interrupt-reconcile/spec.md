## MODIFIED Requirements

### Requirement: Transcript 提取邏輯共用化

系統 SHALL 將 `extractFromTranscriptAtOffset`、`extractSubagentTokens` 等提取函式移至 `internal/transcript` package，供 `cmd/tt/record.go` 與 `internal/reconcile/reconcile.go` 共用。此提取邏輯在解析對話紀錄時 SHALL 具備對損毀 JSON 與空行的容錯能力，並能支援最大 1MB 的對話行解析。

#### Scenario: record.go 使用共用提取函式

- **WHEN** `cmd/tt/record.go` 在 Stop hook 觸發時計算 token
- **THEN** 呼叫 `internal/transcript.ExtractWindow`（或等效函式），行為與重構前一致，現有測試全數通過

#### Scenario: reconcile 使用共用提取函式

- **WHEN** `internal/reconcile/reconcile.go` 補算懸空 turn
- **THEN** 呼叫 `internal/transcript.ExtractWindow`，不依賴 cmd 層的任何 context

#### Scenario: Transcript 解析容錯

- **WHEN** 呼叫 `internal/transcript.ExtractWindow` 且對話紀錄包含空行、空白字元行或損毀的 JSON 行
- **THEN** 系統跳過這些無效行，並正確解析其餘有效對話 entries，不拋出錯誤或陷入無窮迴圈

#### Scenario: Transcript 超長單行支援

- **WHEN** 對話紀錄中單行大小介於 64KB 與 1MB 之間
- **THEN** 系統仍能正常讀取與解析該行並統計 token，不因緩衝區限制而報錯
