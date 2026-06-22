## Context

`tt report` 和 `tt serve` 指令會呼叫 `reconcile.MaybeReconcile`，此過程會呼叫 `loadTranscript` 來讀取並解析 JSONL 格式的對話紀錄。
目前的 `loadTranscript` 實作使用 Go 標準庫的 `json.Decoder` 與 `dec.More()` 迴圈進行解析。若遇到檔案尾端空行、損毀字元或不完整行，`Decode` 方法會回傳錯誤，但其讀取位移不會前進。這導致 `dec.More()` 持續回傳 `true`，使得解析器陷入 CPU 100% 的無窮迴圈，程式因而卡死。

## Goals / Non-Goals

**Goals:**
- 解決 `loadTranscript` 在讀取異常對話紀錄時陷入無窮迴圈的問題，確保指令能穩定執行完畢。
- 支援最大 1MB 的單行 JSONL 記錄，以處理包含長 tool 輸出或大 payload 的 entries。
- 提高解析的容錯能力，對於空行、空白行或語法損毀的 JSON 行進行跳過，不中斷其後續有效對話記錄的解析。

**Non-Goals:**
- 不修改對話紀錄檔案本身的內容。
- 不變更 `ExtractWindow` 的外部介面與回傳型別。
- 不引入外部第三方 JSON 解析庫。

## Decisions

### 1. 改用 `bufio.Scanner` 搭配 `json.Unmarshal`
- **方案**：以 `bufio.Scanner` 逐行讀取檔案，並對每一行呼叫 `json.Unmarshal` 解析為對話 entry。
- **理由**：`bufio.Scanner.Scan()` 會在每一次迭代中確保讀取指標往前移動一行，即使該行格式有誤或為空行，也不會陷入無窮迴圈，從根本上解決卡死問題。
- **替代方案**：繼續使用 `json.Decoder` 但在偵測到 error 時手動尋找下一個 newline 字元推進位移。此方案實作複雜且易出錯，因此不予採用。

### 2. 配置 1MB 上限的讀取緩衝區
- **方案**：使用 `sc.Buffer(make([]byte, 64*1024), 1024*1024)` 設定 `bufio.Scanner` 的緩衝區。
- **理由**：`bufio.Scanner` 的預設緩衝區上限為 64KB。在 AI 對話紀錄中，部分 tool 輸出（例如讀取大量程式碼或複雜的 schema）單行長度可能輕易超過 64KB。若不調大上限，Scanner 將在掃描該行時拋出 `ErrTooLong` 並中斷讀取。1MB 的緩衝上限符合專案在 `insights.md` 中規定的最佳實踐。

### 3. 靜默跳過無效行
- **方案**：若 `json.Unmarshal` 解析失敗，或該行經 `bytes.TrimSpace` 後長度為 0，則 `continue` 讀取下一行。
- **理由**：AI 工具在寫入 JSONL 時可能因為中斷而留下不完整行，或尾端殘留空行。靜默跳過無效行能提供最大容錯度，避免因為單行損毀而導致整個對話的 token 統計失效。

## Risks / Trade-offs

- **[Risk]** 超大對話行（超過 1MB）仍會導致 `bufio.Scanner` 拋出 `ErrTooLong`。
  - *Mitigation*：1MB 對於本專案的 context size 和 logs 來說已足夠大，且本設計在 `sc.Err()` 時會正確回傳錯誤，不致於掛起。
- **[Risk]** 跳過損毀的 JSON 行可能導致 token 統計略低於實際消耗。
  - *Mitigation*：相比於程式卡死，容錯讀取並統計其餘有效 turn 是更可接受的折衷方案。
