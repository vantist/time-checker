## Context

在目前的系統設計中，`tt` 在記錄 AI 工具的 Token 與費用時，將該 turn 中所有的 Token（包含主 Agent 與 Subagent）合併累加至 `turns` 表的單一欄位中，並使用「主 Agent 的 Model」價格計算費用。當主客角色使用不同 model 時（例如主 Agent 為 Claude 3.5 Sonnet，Subagent 使用 Claude 3.5 Haiku），會導致費用被高估。另外，現有的 `turns` 表缺乏多 model 明細欄位，無法滿足主客佔比與明細查詢的需求。

## Goals / Non-Goals

**Goals:**
- 提供精確的多 Model 費用計算，區分主 Agent 與 Subagent。
- 新增明細關聯表 `turn_model_usages` 儲存每個 turn 內各個 model 的 Token 與費用資訊。
- 提供資料庫 migration 機制，將既有 `turns` 資料無痛 backfill 至新表。
- 在 CLI `tt report` 與網頁 Dashboard (`tt serve`) 呈現 By Model & Role 的統計明細。

**Non-Goals:**
- 修改 AI 工具 hook 的基本調用格式或輸入格式。
- 修改 `turns` 主表既有欄位（保留作為預先聚合快取，以相容既有報表查詢）。

## Decisions

### 1. 混合雙軌資料模型設計
引進獨立的 `turn_model_usages` 關聯表，以 `(turn_id, model, is_subagent)` 為複合主鍵。
- **Rationale**:
  - 保留 `turns` 主表原有 Token 與估計費用欄位作為快取，能避免重構所有既有查詢（如專案總覽、歷史趨勢等），確保效能。
  - `turn_model_usages` 細分記錄能支援多 Model 精確計費，並支援主客角色 (Main vs Subagent) 的佔比分析。

#### Schema 定義:
```sql
CREATE TABLE IF NOT EXISTS turn_model_usages (
    id                          INTEGER PRIMARY KEY AUTOINCREMENT,
    turn_id                     INTEGER NOT NULL REFERENCES turns(id) ON DELETE CASCADE,
    model                       TEXT NOT NULL,
    is_subagent                 BOOLEAN NOT NULL DEFAULT 0,
    input_tokens                INTEGER NOT NULL DEFAULT 0,
    output_tokens               INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens           INTEGER NOT NULL DEFAULT 0,
    cache_creation_tokens       INTEGER NOT NULL DEFAULT 0,
    cache_creation_5m_tokens    INTEGER NOT NULL DEFAULT 0,
    cache_creation_1h_tokens    INTEGER NOT NULL DEFAULT 0,
    estimated_cost_usd          REAL NOT NULL DEFAULT 0.0,
    UNIQUE(turn_id, model, is_subagent)
);
CREATE INDEX IF NOT EXISTS idx_turn_model_usages_turn_id ON turn_model_usages(turn_id);
```

### 2. 資料庫初始化與歷史資料 Backfill
在 `internal/db/schema.go` 的 `migrate` 流程中新增 `turn_model_usages` 建立，並執行 Backfill SQL。
- **Rationale**: 透過一次性的 Insert Select 將歷史 `turns` 紀錄依據其 `turns.model` 或 `sessions.model` 無痛轉換為 `turn_model_usages` 的第一筆 (is_subagent = 0) 紀錄，確保系統升級後報表的一致性。

### 3. Transcript 解析與費用計算重構
修改 `internal/transcript` 中的 `WindowResult` 結構與解析邏輯，使其返回 `[]ModelUsage`：
```go
type ModelUsage struct {
	Model               string
	IsSubagent          bool
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int
	CacheCreation5m     int
	CacheCreation1h     int
	EstimatedCostUSD    float64
}
```
- **Rationale**:
  - `reconcile` 與 `recorder/response.go` 讀取 `[]ModelUsage` 後，遍歷每一筆資料並根據其 Model 計算對應費用。
  - 將所有 `ModelUsage` 寫入 `turn_model_usages` 表，同時將其加總值更新至 `turns` 主表對應欄位，實現 pre-aggregation 快取。

### 4. 報表與 Dashboard 呈現
- **CLI Report (`tt report`)**:
  - 新增 `─── By Model & Role ───` 區塊，列出各 Model 以及 Main/Subagent 的 Token 與費用小計。
- **Web Dashboard (`tt serve`)**:
  - `report.HandleAPIReport` 回傳的 JSON 結構中新增 model usages 統計。
  - Dashboard HTML 頁面新增 Model 分佈比例（圓餅圖或橫條圖）與角色佔比，便於使用者直觀了解各模型花費。

## Risks / Trade-offs

- **[Risk]** Migration 時若既有 turns 數量非常龐大，可能會導致啟動延遲。
  - *Mitigation* -> SQLite 查詢在數十萬筆以內此 Backfill 語法耗時極低 (小於 100ms)，並使用 `NOT EXISTS` 確保不重複匯入。
- **[Risk]** 同時寫入 `turns` 與 `turn_model_usages` 可能存在交易一致性問題。
  - *Mitigation* -> 確保 `recorder` 與 `reconcile` 在寫入時使用 SQLite transaction 以保證資料完整性。
