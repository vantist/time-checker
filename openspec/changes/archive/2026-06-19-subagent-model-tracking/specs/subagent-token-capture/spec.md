## MODIFIED Requirements

### Requirement: 提取 subagent token 並合入 turn 成本

系統 SHALL 在執行 `tt record response` 或 `reconcile` 時，掃描主 transcript 中 `[from, to)` 範圍內的 `tool_use` entries，找出 `name == "Agent"` 的呼叫，透過對應的 `meta.json` 找到 subagent jsonl，提取 subagent 的 model 與 token 消耗，並以 `is_subagent = 1` 歸屬。

#### Scenario: 無 subagents 目錄時回傳零

- **WHEN** 主 transcript 同層不存在 `<session_id>/subagents/` 目錄
- **THEN** `ExtractWindow` 只回傳主 Agent 的 model usage，不包含 subagent 的 usage

#### Scenario: Agent tool_use 有對應 meta.json 時合計 token

- **WHEN** 主 transcript `[from, to)` 範圍內有 `tool_use { name: "Agent", id: "toulu_xxx" }`，且 `subagents/agent-yyy.meta.json` 的 `toolUseId == "toulu_xxx"`，且 `subagents/agent-yyy.jsonl` 有 assistant entries
- **THEN** `ExtractWindow` 回傳結果中包含一個獨立的 `ModelUsage`，標記為 `is_subagent = true`，記錄該 subagent 的 model 與 token 消耗小計

#### Scenario: 多個 subagent 時合計所有匹配的 token

- **WHEN** 主 transcript `[from, to)` 範圍內有多個 `tool_use { name: "Agent" }`，各自有對應 meta.json 和 jsonl
- **THEN** `ExtractWindow` 對相同 model 的 subagents 進行合併累加，回傳每個 model 對應的 `is_subagent = true` 總計

#### Scenario: to 邊界之後的 Agent tool_use 不被計入

- **WHEN** 主 transcript 第 `to` 行之後存在 Agent tool_use entries（屬於後續 turn）
- **THEN** 提取邏輯不收集第 `to` 行（含）之後的 Agent ID，不計算這些 subagent 的 token

#### Scenario: meta.json 的 toolUseId 不在本 turn 的 Agent 呼叫中

- **WHEN** subagents 目錄有 meta.json，但其 `toolUseId` 對應的 tool_use 在 offset 之前（前一個 turn）或在 `to` 之後
- **THEN** 提取邏輯不計算該 subagent 的 token

#### Scenario: subagent jsonl 不存在時略過

- **WHEN** meta.json 存在且 toolUseId 匹配，但對應的 `.jsonl` 檔案不存在
- **THEN** 提取邏輯略過該 subagent，繼續處理其他 subagent

#### Scenario: subagent token 合入 WindowResult

- **WHEN** `ExtractWindow` 完成主 transcript 提取，且找到匹配的 subagent 消耗
- **THEN** 最終回傳的 `WindowResult.Usages` 同時包含主 Agent (`is_subagent = false`) 與 subagent (`is_subagent = true`) 的詳細列表

### Requirement: ExtractWindow 回傳 typed struct

系統 SHALL 提供 `transcript.WindowResult` struct 與 `ModelUsage` struct 作為 `ExtractWindow` 與 `ExtractLastTurn` 的回傳型別，取代 JSON string 傳遞。

```go
type WindowResult struct {
    Usages []ModelUsage
}

type ModelUsage struct {
    Model               string
    IsSubagent          bool
    InputTokens         int
    OutputTokens        int
    CacheReadTokens     int
    CacheCreationTokens int
    CacheCreation5m     int
    CacheCreation1h     int
}
```

#### Scenario: ExtractWindow 回傳 WindowResult

- **WHEN** `transcript.ExtractWindow(path, from, to)` 被呼叫，transcript 中有 assistant entries
- **THEN** 回傳 `(WindowResult, error)`，所有欄位填入對應值，`error = nil`

#### Scenario: 空 transcript 回傳零值 WindowResult

- **WHEN** `transcript.ExtractWindow` 讀到的 window 無 any assistant entry
- **THEN** 回傳 `(WindowResult{}, nil)`（零值 struct，非 error）
