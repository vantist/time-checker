# Subagent Model Tracking and Redesign

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

在目前的設計中，`tt` 當呼叫 subagent（如 `/spex-brainstorming`）時，雖然能夠透過 reconcile 擷取 subagent 的 Token，但是：
1. **費用計算錯誤**：所有的 Token（包含主 Agent 與 Subagent）均直接累加至該 turn 的總量中，並統一使用「主 Agent 的 Model 價格」計費。若主 Agent 使用 Sonnet 而 Subagent 使用 Haiku，會造成費用被大幅高估。
2. **Schema 限制**：資料庫 `turns` 表與 `sessions` 表各自僅有單一 `model` 欄位，無法細分紀錄單次 Turn 中多個 model 的使用明細與主客佔比。

## Decision

引進獨立的 `turn_model_usages` 關聯表，以 `(turn_id, model, is_subagent)` 為複合主鍵，細分記錄每個 model 的 Token 與費用。`turns` 主表仍保留 pre-aggregated 的合計值，實現雙軌混合架構。

## Rationale

這個設計折衷了「資料庫正規化」與「查詢性能/相容性」。
* 保留 `turns` 主表的總和快取：能使既有大盤、專案彙整與 API 查詢在改動最小的情況下運作，效能最好。
* 引進 `turn_model_usages`：能以最精確的 Model 單價計算費用，並可分析主客角色（Main vs Subagent）的開銷比例。

## Approach

採用**混合雙軌方案**。
1. **Schema 擴充**：
   - 新增 `turn_model_usages` 表與 `(turn_id, model, is_subagent)` 唯一約束。
   - 在 DB 初始化加入 migration backfill SQL，將歷史 turns 資料無痛匯入新表。
2. **提取/計算重構**：
   - 修改 `internal/transcript` 的 `WindowResult` 結構，回傳 `[]ModelUsage` 陣列。
   - `reconcile.go` 與 `recorder/response.go` 中，根據各 model 個別計算費用，寫入明細，並將總和同步更新至 `turns` 主表。
3. **報表呈現**：
   - `report.Query` 增量查詢明細，並在 CLI `report` 輸出加入 `─── By Model & Role ───` 統計表。
   - 在 `tt serve` 網頁 Dashboard 新增 By Model & Role 的折線/比例與表格區塊。

## Design Notes

### Database Migration Details
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

### Backfill Query
```sql
INSERT INTO turn_model_usages (
    turn_id, model, is_subagent, 
    input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, 
    cache_creation_5m_tokens, cache_creation_1h_tokens, estimated_cost_usd
)
SELECT 
    t.id, 
    COALESCE(NULLIF(t.model, ''), NULLIF(s.model, ''), 'unknown'), 
    0,
    COALESCE(t.input_tokens, 0), 
    COALESCE(t.output_tokens, 0), 
    COALESCE(t.cache_read_tokens, 0), 
    COALESCE(t.cache_creation_tokens, 0),
    COALESCE(t.cache_creation_5m_tokens, 0), 
    COALESCE(t.cache_creation_1h_tokens, 0), 
    COALESCE(t.estimated_cost_usd, 0.0)
FROM turns t
JOIN sessions s ON s.id = t.session_id
WHERE (t.input_tokens IS NOT NULL OR t.output_tokens IS NOT NULL)
  AND t.id NOT IN (SELECT DISTINCT turn_id FROM turn_model_usages);
```

## Insights to Capture

- `design.md`: 說明 subagent 多 model 計費設計與 `turn_model_usages` 雙軌機制。
- `specs/event-recording/spec.md`: 新增有關 multiple models per turn 與 is_subagent 歸屬的規格描述。
- `proposal.md`: 將此變更納入新 change 的範疇。
- `tasks.md`: 分解 Migration、Transcript 重構、Reconcile 寫入、Report 統計、Dashboard UI 等具體任務。

## Open Questions

無。
