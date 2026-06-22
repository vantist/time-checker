## MODIFIED Requirements

### Requirement: 基本報表輸出

系統 SHALL 透過 `tt report` 命令輸出過去 7 天（預設）的聚合統計，格式為純文字，包含：
- Sessions 數量
- Agent 時間（h m 格式）
- User 主動時間（h m 格式，含使用的 idle threshold）
- Token 總量（包含 Input, Output, Cache read, Cache create 欄位）
- 預估成本（USD）
- 各個 Model 以及其角色 (Main/Subagent) 的 Token 與預估費用明細統計表

#### Scenario: 無資料時顯示空報表而非錯誤

- **WHEN** `tt report` 被呼叫，且資料庫中 7 天內無任何 turns
- **THEN** 輸出 "No data for the selected period."
- **THEN** exit code 0

#### Scenario: 有資料時輸出格式正確

- **WHEN** 7 天內有資料，`tt report` 被呼叫
- **THEN** stdout 輸出包含 "Sessions:", "Agent time:", "User active:", "Input:", "Output:", "Cache read:", "Cache create:", "Est. cost:" 等欄位
- **THEN** Agent time 格式為 `Xh Ym`（如 `2h 34m`）
- **THEN** 包含 "─── By Model & Role ───" 標題，下方列出各模型及角色的 input, output, cache read, cache create token 與費用明細

### Requirement: 篩選條件

系統 SHALL 支援以下篩選選項：

| 選項 | 說明 |
|------|------|
| `--project <name>` | 依 `sessions.project` 路徑末段或完整路徑篩選 |
| `--since <duration\|date>` | 時間範圍：`7d`、`30d`、`2026-06-01` |
| `--format json` | 輸出 JSON 格式（預設 text） |

#### Scenario: --project 篩選只顯示指定專案

- **WHEN** `tt report --project time-tracker` 被呼叫
- **THEN** 只包含 `sessions.project` 路徑含 "time-tracker" 的 sessions

#### Scenario: --since 7d 篩選過去 7 天

- **WHEN** `tt report --since 7d` 被呼叫（今日為 2026-06-17）
- **THEN** 只包含 `prompt_at >= 2026-06-10 00:00:00 UTC` 的 turns

#### Scenario: --since 指定日期篩選

- **WHEN** `tt report --since 2026-06-01` 被呼叫
- **THEN** 只包含 `prompt_at >= 2026-06-01 00:00:00 UTC` 的 turns

#### Scenario: --format json 輸出合法 JSON

- **WHEN** `tt report --format json` 被呼叫
- **THEN** stdout 為合法 JSON，可被 `jq` 解析
- **THEN** JSON 包含 `sessions_count`, `agent_time_sec`, `user_active_time_sec`, `input_tokens`, `output_tokens`, `cache_read_tokens`, `cache_creation_tokens`, `estimated_cost_usd` 欄位
- **THEN** JSON 中包含 `model_usages` 陣列，詳細列出各 model 的 Token (含 input, output, cache_read, cache_creation) 與費用小計，且區分 `is_subagent`
