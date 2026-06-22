# Integration Testing for Unpushed CLI Features

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

本地已有多個未推送的實作功能（包括分支修復、主動 preempt 搶占、15 分鐘空閒超時以及多工具 log 提取與 fallback）。為了驗證這些功能在真實 CLI 呼叫、環境變數隔離和 SQLite 資料庫層面是否正確動作，我們需要設計並實作一個 Go 的端到端整合測試套件 (`cmd/tt/integration_test.go`)。

## Decision

採用 **子進程命令列執行法（Black-Box Integration Test）** 來實作此整合測試。測試執行時會先編譯出臨時 `tt` 二進位檔，並在執行 CLI 時完全覆寫 `HOME`、`TT_DB_PATH` 及 `PROCESS_PID/PROCESS_START` 以實現環境隔離與行為模擬。最後直接透過 `database/sql` 讀取臨時 SQLite 檔案來進行斷言驗證。

## Rationale

- **真實行為模擬**：子進程執行法能完整模擬 stdin 讀取、進程生命週期判斷 (`IsAlive`) 以及 CLI 啟動時的檔案鎖（File Lock）與初始化邏輯。
- **防止全域污染**：隔離的子進程能避免 Cobra commands 全域 flag 殘留導致的測試干擾。
- **高覆蓋度**：涵蓋對多個工具（Claude Code, Copilot CLI, Google Antigravity）各自不同的 stdin JSON 與 log 檔案的端到端解析與儲存。

## Approach

1. 在 `cmd/tt/integration_test.go` 中實作動態編譯臨時 `tt` 二進位檔及清理的 `TestMain` 或 `t.Cleanup` 腳手架。
2. 建立 `runTT` 輔助函式，封裝 `exec.Command` 並設定環境變數 `HOME=<temp>` 與 `TT_DB_PATH=<temp>/test.db`。
3. 實作以下 5 個整合測試場景：
   - **TestIntegration_GitBranchRepair**：在 Git 倉庫下寫入無 branch 紀錄，驗證 `reconcile` 自動修復。
   - **TestIntegration_ActiveTurnPreemption**：連續錄製兩次 prompt 驗證前一次 active turn 被自動 pre-empt 關閉。
   - **TestIntegration_IdleThresholdReconcile**：寫入 20 分鐘前 dangling turn，驗證被依 15 分鐘閾值自動 reconcile。
   - **TestIntegration_FallbackDefaultModel**：以缺省欄位錄製，驗證自動 fallback 當前路徑與預設模型。
   - **TestIntegration_MultiToolIntegration**：模擬不同工具的 stdin payload 與 mock 紀錄檔，驗證各自的寫入與 token 解析行為。

## Design Notes

### 整合測試腳手架與 `runTT` 範例

```go
func TestMain(m *testing.M) {
	// 1. go build -o <temp_dir>/tt ./cmd/tt
	// 2. 執行測試 m.Run()
	// 3. 清理臨時檔案
}

func runTT(t *testing.T, tempDir string, stdinJSON string, args ...string) (string, string, error) {
	cmd := exec.Command(filepath.Join(tempDir, "tt"), args...)
	cmd.Env = append(os.Environ(),
		"HOME="+tempDir,
		"TT_DB_PATH="+filepath.Join(tempDir, "test.db"),
	)
	if stdinJSON != "" {
		cmd.Stdin = strings.NewReader(stdinJSON)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
```

### 多工具 Mock 紀錄檔格式與 Stdin Payload

1. **Claude Code**
   - **Stdin**: `{"session_id": "c-1", "cwd": "<dir>", "hook_event_name": "UserPromptSubmit", "model": "claude-sonnet-4-6"}`
   - **Transcript JSONL**: `{"type":"assistant","message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":100,"output_tokens":50}}}`

2. **Copilot CLI**
   - **Stdin**: `{"sessionId": "cp-1", "cwd": "<dir>", "timestamp": 1700000000000}`
   - **Transcript JSONL (events.jsonl)**: `{"type":"session.shutdown","data":{"mainModel":"gpt-4o","modelMetrics":{"gpt-4o":{"usage":{"inputTokens":120,"outputTokens":60}}}}}`

3. **Google Antigravity**
   - **Stdin**: `{"conversationId": "ag-1", "transcriptPath": "<path>"}`
   - **Transcript JSONL**: 與 Claude Code 同樣的 `entry` 結構。

### 資料庫驗證斷言

```go
func assertSession(t *testing.T, db *sql.DB, sessionID string, wantTool string, wantBranch string) {
	var tool, branch sql.NullString
	err := db.QueryRow("SELECT tool, branch FROM sessions WHERE id=?", sessionID).Scan(&tool, &branch)
	if err != nil {
		t.Fatalf("query session %s: %v", sessionID, err)
	}
	if tool.String != wantTool {
		t.Errorf("session tool = %q, want %q", tool.String, wantTool)
	}
	if wantBranch != "" && branch.String != wantBranch {
		t.Errorf("session branch = %q, want %q", branch.String, wantBranch)
	}
}
```

## Insights to Capture

- `design.md`: 說明各工具 stdin 及日誌檔案（events.jsonl vs transcript.jsonl）的整合測試細節。
- `tasks.md`: 新增整合測試的具體任務。

## Open Questions

None.
