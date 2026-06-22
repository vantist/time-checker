# resolve-report-serve-hang

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

當使用者執行 `tt report` 或 `tt serve` 時，指令會卡死且沒有任何回應。

經調查，這是因為兩者都會執行 `reconcile.MaybeReconcile`，其中會呼叫 `loadTranscript` 讀取 JSONL 對話紀錄。原先的實作使用 `json.Decoder` 與 `dec.More()` 迴圈，若遇到檔案尾端空行、損毀字元或正在寫入的不完整行，`Decode` 會拋出 error 且不前進位移，導致 `dec.More()` 持續回傳 `true`，陷入無窮迴圈。

先前在 `.spex/insights.md` 中也提到了這個已知問題（使用 `bufio.Scanner` 並設置 1MB 緩衝區來解決此問題的建議）。

## Decision

改用 `bufio.Scanner` + `json.Unmarshal` 替代 `json.Decoder` 逐行解析對話紀錄，並顯式配置最大 1MB 的讀取緩衝區，在遇到無效 JSON 或空行時直接跳過，以提升解析強健性並避免程式卡死。

## Rationale

* 採用 `bufio.Scanner` 能確保遇到語法錯誤或空行時，指標仍能正確前進到下一行，消除無窮迴圈的隱患。
* 配置 1MB 上限（`sc.Buffer(make([]byte, 64*1024), 1024*1024)`）是為了解決 `bufio.Scanner` 預設 64KB 緩衝區可能因大對話行（如包含複雜 tool 輸出）而提前中斷的問題。這符合專案在 `insights.md` 中的最佳實踐規範。

## Approach

修改 `internal/transcript/extract.go` 中的 `loadTranscript`：
1. 建立 `bufio.NewScanner` 並設定 1MB 緩衝區限制。
2. 逐行讀取，使用 `bytes.TrimSpace` 過濾空行。
3. 對非空行呼叫 `json.Unmarshal`。
4. 若 unmarshal 失敗，僅跳過該行而不中斷整個檔案的讀取，以對部分毀損的對話紀錄保持最大的容錯。

同時在 `internal/transcript/extract_test.go` 中新增單元測試，驗證遇到空行與受損 JSON 時的過濾與正確解析能力。

## Design Notes

### 1. `internal/transcript/extract.go` 程式碼調整
```go
func loadTranscript(path string) ([]entry, error) {
	f, err := os.Open(expandHome(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var all []entry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)

	for sc.Scan() {
		line := sc.Bytes()
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var e entry
		if err := json.Unmarshal(line, &e); err != nil {
			// 跳過出錯行，不中斷
			continue
		}
		all = append(all, e)
	}

	if err := sc.Err(); err != nil {
		return nil, err
	}
	return all, nil
}
```

### 2. `internal/transcript/extract_test.go` 單元測試新增
```go
func TestExtractWindow_CorruptAndTrailingLines(t *testing.T) {
	lines := []string{
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-3-5-sonnet","usage":{"input_tokens":10,"output_tokens":5}}}`,
		``,
		`{invalid json`,
		`   `,
		`{"type":"user","isSidechain":false}`,
		`{"type":"assistant","isSidechain":false,"message":{"model":"claude-3-5-sonnet","usage":{"input_tokens":20,"output_tokens":10}}}`,
		` `,
	}
	path := writeLines(t, lines)

	result, err := transcript.ExtractWindow(path, 0, -1)
	if err != nil {
		t.Fatalf("ExtractWindow failed: %v", err)
	}

	if len(result.Usages) != 1 {
		t.Fatalf("expected 1 usage grouping, got %d: %+v", len(result.Usages), result.Usages)
	}

	u := result.Usages[0]
	if u.InputTokens != 30 || u.OutputTokens != 15 {
		t.Errorf("expected usage (30, 15), got (%d, %d)", u.InputTokens, u.OutputTokens)
	}
}
```

## Insights to Capture

- `design.md`: 遇到 JSONL 日誌讀取掛起問題時，優先檢查是否在 `dec.More()` 迴圈中缺乏錯誤推進。
- `tasks.md`:
  - 任務 1.1: 在 `internal/transcript/extract.go` 重構 `loadTranscript` 改為 `bufio.Scanner` 與 `json.Unmarshal`。
  - 任務 1.2: 在 `internal/transcript/extract_test.go` 新增 `TestExtractWindow_CorruptAndTrailingLines` 測試並執行 `go test ./...` 驗證其正確性。

## Open Questions

無。已達成完全收斂。
