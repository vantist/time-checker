## Why

當使用者執行 `tt report` 或 `tt serve` 時，指令會卡死且沒有任何回應。這是因為這兩個指令在讀取 JSONL 對話紀錄時，如果遇到檔案尾端的空行、損毀字元或正在寫入的不完整行，`json.Decoder` 與 `dec.More()` 迴圈會陷入無窮迴圈。

## What Changes

1. 修改對話紀錄讀取函式 `loadTranscript` 的解析實作：
   - 使用 `bufio.Scanner` 與 `json.Unmarshal` 替代目前的 `json.Decoder` 進行逐行解析，消除因語法錯誤或空行導致位移不前進的隱患。
   - 配置最大 1MB 的讀取緩衝區，支援大型對話（包含大量/複雜 tool 輸出的對話行），避免因超過預設 64KB 限制而中斷。
   - 對於損毀 Rar/JSON 行或空行採取容錯機制，在 `json.Unmarshal` 失敗或空行時跳過該行並繼續處理，不中斷整個檔案讀取。
2. 新增單元測試：
   - 在 `internal/transcript/extract_test.go` 中新增測試，驗證遇到空行、空白字元及受損 JSON 時的容錯過濾與正確解析能力。

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- interrupt-reconcile: 擴充 Transcript 提取邏輯，增加對損毀 JSON 行與空白行的容錯解析，並配置 1MB 讀取緩衝區以防長行解析失敗。

## Impact

- Affected code:
  - Modified:
    - internal/transcript/extract.go
    - internal/transcript/extract_test.go

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-22-brainstorm-report-hang.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
