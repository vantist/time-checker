# session-management Specification

## Purpose
TBD - created by archiving change ai-tool-time-tracker. Update Purpose after archive.
## Requirements
### Requirement: Session 資料模型

系統 SHALL 維護 `sessions` 表，欄位如下：

| 欄位 | 類型 | 說明 |
|------|------|------|
| `id` | TEXT PRIMARY KEY | AI 工具提供的 session ID |
| `project` | TEXT | git root 路徑，fallback 為 cwd |
| `tool` | TEXT | `"claude-code"` 或 `"copilot-cli"` |
| `started_at` | INTEGER | unix milliseconds，第一個 prompt 時間 |
| `ended_at` | INTEGER | unix milliseconds，最後一個 response 時間，可 NULL |
| `branch` | TEXT | git branch 名稱，可 NULL |
| `work_item` | TEXT | 手動標記的工作項目，可 NULL |

#### Scenario: Session upsert 保留 started_at

- **WHEN** `sessions` 表已存在 `id = "abc123"` 的 session（`started_at = T1`），再次 INSERT 相同 ID
- **THEN** `started_at` 保持 T1，不被覆蓋
- **THEN** `ended_at` 更新為最新值

### Requirement: Turn 資料模型

系統 SHALL 維護 `turns` 表，欄位如下：

| 欄位 | 類型 | 說明 |
|------|------|------|
| `id` | INTEGER PRIMARY KEY AUTOINCREMENT | |
| `session_id` | TEXT REFERENCES sessions | |
| `model` | TEXT | 模型名稱 |
| `prompt_at` | INTEGER | unix milliseconds |
| `response_at` | INTEGER | unix milliseconds，可 NULL（等待 response） |
| `input_tokens` | INTEGER | 可 NULL |
| `output_tokens` | INTEGER | 可 NULL |
| `cache_read_tokens` | INTEGER | 可 NULL |
| `cache_creation_tokens` | INTEGER | 可 NULL |
| `estimated_cost_usd` | REAL | 可 NULL（未知 model 時） |

#### Scenario: 資料庫初始化建立 schema

- **WHEN** `tt` 首次執行任何命令，且 `~/.tt/data.db` 不存在
- **THEN** 系統自動建立 `~/.tt/data.db`
- **THEN** 建立 `sessions` 表與 `turns` 表（包含所有欄位與外鍵約束）
- **THEN** 命令正常繼續執行

#### Scenario: 已存在的資料庫不重建 schema

- **WHEN** `~/.tt/data.db` 已存在且 schema 正確
- **THEN** 系統不刪除或重建任何表

### Requirement: 資料庫路徑可透過環境變數覆蓋

系統 SHALL 在環境變數 `TT_DB_PATH` 設定時，使用該路徑作為 SQLite 資料庫路徑（取代預設的 `~/.tt/data.db`）。

#### Scenario: 測試時使用臨時資料庫

- **WHEN** 環境變數 `TT_DB_PATH=/tmp/tt-test.db` 設定
- **THEN** 所有讀寫操作使用 `/tmp/tt-test.db`，不碰 `~/.tt/data.db`

### Requirement: 自動修補歷史 Session 的分支與中繼資料

系統在執行 `reconcile` 前，SHALL 自動修復歷史 Session 中缺失的專案路徑 (`project`)、模型名稱 (`model`) 與 Git 分支名稱 (`branch`)。

#### Scenario: 歷史 Session 的 Git 分支修復成功
- **WHEN** Session 的 `branch` 為空（或為空字串），且專案路徑 `project` 為有效的 Git 專案目錄
- **THEN** 系統自動解析並修復其 `branch` 欄位為該專案目前之 Git 分支名稱

#### Scenario: 非 Git 專案歷史 Session 的分支標記為佔位符
- **WHEN** Session 的 `branch` 為空，且專案路徑 `project` 為非 Git 目錄（或解析失敗）
- **THEN** 系統自動修復並標記該 `branch` 欄位為 "-"，以防止後續重複解析

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

