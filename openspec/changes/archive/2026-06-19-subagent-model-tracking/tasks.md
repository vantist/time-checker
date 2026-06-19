## 1. 資料庫 Schema 與 Migration

- [x] 1.1 在 `internal/db/schema.go` 中新增 `turn_model_usages` 關聯表
- [x] 1.2 在 `internal/db/schema.go` 的 `migrate` 函式中新增歷史資料 backfill 邏輯
- [x] 1.3 在 `internal/db/schema_test.go` 撰寫測試，驗證遷移與歷史資料 backfill 的正確性

## 2. Transcript 解析重構 (TDD)

- [x] 2.1 在 `internal/transcript/transcript.go` 中定義 `ModelUsage` 與修改 `WindowResult` 結構
- [x] 2.2 在 `internal/transcript/transcript_test.go` 新增多模型與 subagent token 提取的測試案例 (撰寫失敗測試)
- [x] 2.3 修改 `internal/transcript/transcript.go` 的解析邏輯以通過測試

## 3. Recorder 與 Reconcile 寫入邏輯重構 (TDD)

- [x] 3.1 調整定價模組 `internal/pricing` 以支援對單一 `ModelUsage` 計算費用
- [x] 3.2 於 `internal/recorder/response_test.go` 與 `internal/reconcile/reconcile_test.go` 撰寫測試，驗證寫入/重算時對 `turn_model_usages` 與 `turns` 的更新行為 (撰寫失敗測試)
- [x] 3.3 修改 `internal/recorder/response.go`，將 response 的 model usage 寫入關聯表並同步累加至 `turns` 表
- [x] 3.4 修改 `internal/reconcile/reconcile.go`，在重算時替換 `turn_model_usages` 表中該 turn 的舊明細，並同步更新至 `turns` 表

## 4. CLI 報表功能擴充

- [x] 4.1 在 `internal/report/report.go` 新增查詢 `turn_model_usages` 明細的 SQL 邏輯
- [x] 4.2 修改 `tt report` 的純文字輸出，新增 `─── By Model & Role ───` 統計區塊
- [x] 4.3 修改 `tt report --format json` 輸出，在 JSON payload 中新增 `model_usages` 陣列
- [x] 4.4 撰寫/更新 `internal/report/report_test.go` 的測試案例，驗證報表統計與輸出的正確性

## 5. Web Dashboard 網頁儀表板擴充

- [x] 5.1 修改 `HandleAPIReport` API 端點，回傳結構中新增 `model_usages` 明細
- [x] 5.2 修改 `internal/report/html.go` 中的網頁 Dashboard 範本與 Javascript 邏輯，新增 By Model & Role 的 CSS 百分比比例條與明細表格
- [x] 5.3 撰寫 `internal/report/html_test.go` 測試，驗證 Dashboard 渲染與 API 的輸出欄位
