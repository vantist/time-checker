# setup-expansion

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

目前 `tt setup` 僅支援 `Claude Code` 與 `GitHub Copilot CLI` 的 hook/指令設定。然而，底層與 log 解析實作已能支援 `Google Antigravity` 與 `OpenAI Codex`，這造成了設定面與底層實作能力的不一致，應予擴充。

## Decision

擴充 `tt setup` 指令，新增 `--antigravity` 與 `--codex` 支援，將 hooks 分別以冪等（Idempotent）方式合併至其全域設定檔（`~/.gemini/config/hooks.json` 與 `~/.codex/hooks.json`）。同時，在 `cmd/tt/record.go` 中解析來自 stdin 的 Antigravity 欄位 `conversationId` 與 `transcriptPath`。

## Rationale

1. **模組化共用 Helper (Approach A)**：Claude Code、Antigravity 與 Codex 的 hooks 設定皆為 JSON 格式，且合併與冪等清理邏輯類似。將其抽象化為通用的合併 helper，可極大地減少重複程式碼，且有利於後續擴充。
2. **與現有 API 無縫銜接**：Codex transcript 使用標準的 stdin 傳遞與現有機制相同；Antigravity 僅需在 `hookPayload` 中額外做欄位對應（`conversationId` -> `SessionID`，`transcriptPath` -> `TranscriptPath`），即可自動使用既有的 log 解析器。

## Approach

1. 在 `internal/setup/setup.go` 中實作 `mergeHooksFile` Helper，統一處理讀檔、目錄建立、篩選舊 `_owner == "tt"` 項目、合併與 `0o600` 安全寫入。
2. 重構 `SetupClaudeCode()`、實作 `SetupAntigravity()` 與 `SetupCodex()` 呼叫該 Helper。
3. 在 `cmd/tt/setup_cmd.go` 中加入 `--antigravity` 與 `--codex` flag，並串接上述 Setup 函式。
4. 在 `cmd/tt/record.go` 的 `hookPayload` 中加入 Antigravity 專屬的 `conversationId` 與 `transcriptPath` tags，並在 `readStdinJSON()` 中將其對應並正規化。

## Design Notes

### Antigravity Hook 結構 (`~/.gemini/config/hooks.json`)
```json
{
  "tt": {
    "PreInvocation": [
      {
        "_owner": "tt",
        "type": "command",
        "command": "tt record prompt --tool antigravity"
      }
    ],
    "Stop": [
      {
        "_owner": "tt",
        "type": "command",
        "command": "tt record response --tool antigravity"
      }
    ]
  }
}
```

### Codex Hook 結構 (`~/.codex/hooks.json`)
```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "_owner": "tt",
        "type": "command",
        "command": "tt record prompt --tool codex"
      }
    ],
    "Stop": [
      {
        "_owner": "tt",
        "type": "command",
        "command": "tt record response --tool codex"
      }
    ]
  }
}
```

## Insights to Capture

- `design.md`: 新增 Antigravity 與 Codex 的 Hook 整合與 Payload 欄位規格說明
- `specs/hook-integration/spec.md`: 補充 Antigravity 與 Codex 的 Hook 合併規格要求
- `specs/idempotent-hook-setup/spec.md`: 補充 Setup 寫入 hooks 的冪等性規格
- `proposal.md`: 規劃擴充 `tt setup` 支援以完整支援 Antigravity 及 Codex 
- `tasks.md`: 新增 Helper 重構、SetupAntigravity、SetupCodex、CLI 參數對接及單元測試等任務

## Open Questions

（無，設計已收斂）
