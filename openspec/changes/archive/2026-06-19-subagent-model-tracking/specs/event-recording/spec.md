## ADDED Requirements

### Requirement: turn_model_usages 表新增與歷史資料匯入

系統 SHALL 於資料庫初始化或執行命令時，若發現 `turn_model_usages` 表不存在，則建立該表並執行 backfill SQL 將歷史 `turns` 紀錄依據其 `turns.model` 或 `sessions.model` 匯入該表。

#### Scenario: 首次執行時自動建立 turn_model_usages 表並匯入歷史資料
- **WHEN** 系統啟動，偵測到 `turn_model_usages` 不存在
- **THEN** 建立 `turn_model_usages` 表
- **THEN** 將所有既存且含有 token 數據的 `turns` 紀錄匯入 `turn_model_usages` 作為 `is_subagent = 0` 的歷史紀錄，預設使用 `turns.model` 或 `sessions.model`，若均無則為 `'unknown'`

## MODIFIED Requirements

### Requirement: 記錄 response 事件

系統 SHALL 透過 `tt record response` 子命令接收 hook 呼叫，並更新對應 turn 的 `response_at`、token 欄位（含 `cache_creation_5m_tokens`、`cache_creation_1h_tokens`、`model`）、`estimated_cost_usd`，同時將 `subagent_tokens_settled` 設為 0。此外，系統 MUST 將該 turn 當下的 model token 消耗以 `is_subagent = 0` (主 Agent) 寫入 `turn_model_usages` 關聯表。

命令簽章：
```
tt record response --session <id> --tokens <json>
```

- `--session`：與 `tt record prompt` 相同的 session ID
- `--tokens`：JSON 字串，包含 token 計數（欄位名稱允許多種格式）

#### Scenario: 成功記錄 response 並計算成本

- **WHEN** `tt record response --session abc123 --tokens '{"input_tokens":1000,"output_tokens":200,"cache_read_tokens":500,"cache_creation_tokens":0}'` 被呼叫
- **THEN** 最新一筆 `session_id = "abc123"` 且 `response_at IS NULL` 的 turn，更新 `response_at = 目前 unix ms`
- **THEN** 更新 `input_tokens = 1000`, `output_tokens = 200`, `cache_read_tokens = 500`, `cache_creation_tokens = 0`
- **THEN** 更新 `subagent_tokens_settled = 0`
- **THEN** 根據 turn 的 `model` 查詢定價表，計算並寫入 `estimated_cost_usd`
- **THEN** 在 `turn_model_usages` 表中寫入一筆明細（`turn_id` 為本 turn ID, `model` 為該 turn model, `is_subagent = 0`，以及對應的 token 與預估費用）

#### Scenario: token JSON 欄位名稱容錯

- **WHEN** `--tokens` JSON 使用 `usage.input_tokens` 巢狀格式（`{"usage":{"input_tokens":1000,"output_tokens":200}}`）
- **THEN** 系統正確解析 token 值，功能與扁平格式相同

#### Scenario: token JSON 缺欄位時記錄 NULL

- **WHEN** `--tokens` JSON 缺少 `cache_read_tokens` 欄位
- **THEN** `turns.cache_read_tokens` 寫入 NULL，不報錯，exit code 0

#### Scenario: 找不到對應 prompt turn 時靜默跳過

- **WHEN** `--session abc123` 下找不到 `response_at IS NULL` 的 turn（可能 prompt 記錄失敗）
- **THEN** 命令不報錯，exit code 0，不修改 any 資料
