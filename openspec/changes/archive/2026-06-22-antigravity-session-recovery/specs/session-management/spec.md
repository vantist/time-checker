## ADDED Requirements

### Requirement: 歷史 Session 自動修補

系統 SHALL 在 `MaybeReconcile` 執行時，自動掃描並修補缺失 `project` 或 `model` 欄位的歷史 sessions：
1. 對於匹配的 session，從其 turns 中讀取第一個有效 `transcript_path` 的 transcript 內容。
2. 在 transcript JSON 結構中搜尋包含 Home 目錄的絕對路徑，過濾掉排除名單（`.gemini`, `.claude`, `.copilot`, `Library`, `Downloads`, `Desktop`, `Applications` 等）。
3. 自該路徑向上遞迴尋找 `.git` 或 `go.mod` 來重構專案根目錄；若均無則 fallback 至 `os.Getwd()`。
4. 若 `model` 欄位為空，則解析其 logs、settings.json，或 fallback 至 `gemini-3.5-flash`。
5. 將修補後的 `project` 與 `model` 欄位寫回 DB。

#### Scenario: 成功修補缺失 project 與 model 的 session

- **WHEN** 執行 `MaybeReconcile` 且 DB 中存在 `project` 為空且 `model` 為空的 session，且其 transcript 中含有路徑 `/Users/test/workspace/my-project/file.go`
- **THEN** 系統成功更新該 session，`project` 設為該專案根目錄（如 `/Users/test/workspace/my-project`），`model` 設為 `"gemini-3.5-flash"`
