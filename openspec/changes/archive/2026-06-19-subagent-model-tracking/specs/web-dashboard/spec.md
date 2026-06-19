## ADDED Requirements

### Requirement: 網頁 dashboard 顯示 By Model & Role 佔比與明細

網頁 dashboard (`GET /`) SHALL 包含「By Model & Role」統計區塊，以圖表或比例條（例如純 CSS 百分比條）呈現不同 Model（如 Claude 3.5 Sonnet vs Haiku）與不同角色（Main vs Subagent）的 Token 消耗與費用佔比，並以表格列出各 Model/Role 的明細（包含 Input, Output, Cache, Cost）。

#### Scenario: Dashboard 顯示 By Model & Role 明細
- **WHEN** 瀏覽器請求 `GET /` 且資料庫有包含 subagent 的 token 記錄
- **THEN** 頁面包含 Model 與 Role 的百分比統計條
- **THEN** 頁面包含明細表格，列出各 Model 在不同角色 (Main/Subagent) 下的 Token 與費用

## MODIFIED Requirements

### Requirement: /api/report JSON endpoint

`GET /api/report` SHALL 回傳與 `tt report --json` 相同結構的 JSON，包含 by_project、完整 token 欄位以及 model usages 明細。

#### Scenario: JSON endpoint 回傳正確 Content-Type

- **WHEN** 瀏覽器或 curl 請求 `GET /api/report`
- **THEN** 回應 `Content-Type: application/json`，body 為合法 JSON

#### Scenario: JSON endpoint 欄位完整

- **WHEN** 請求 `GET /api/report`
- **THEN** JSON 含 `sessions`（int）、`agent_time_seconds`（int）、`input_tokens`（int）、`output_tokens`（int）、`cache_read_tokens`（int）、`cache_creation_tokens`（int）、`cost_usd`（float）、`by_project`（陣列）、`daily`（陣列，7 天）
- **THEN** JSON 包含 `model_usages` 陣列，詳細列出各 model 的 Token 與費用小計，且區分 `is_subagent`
