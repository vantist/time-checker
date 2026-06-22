## ADDED Requirements

### Requirement: Google Antigravity Hook 設定

系統 SHALL 在 `tt setup --antigravity` 被呼叫時，在 `~/.gemini/config/hooks.json` 中以 merge 方式加入以下 hooks（若原檔不存在或不含 `tt` 欄位則新建，不覆寫其他已存在的 hooks）：

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

#### Scenario: 首次設定 Google Antigravity hooks

- **WHEN** `tt setup --antigravity` 被呼叫，且 `~/.gemini/config/hooks.json` 不存在或不含 `tt` 欄位
- **THEN** 在 `~/.gemini/config/hooks.json` 中建立 `tt` 屬性，並在其下加入 `PreInvocation` 與 `Stop` hooks
- **THEN** stdout 輸出 `Google Antigravity hooks configured in ~/.gemini/config/hooks.json`

### Requirement: OpenAI Codex Hook 設定

系統 SHALL 在 `tt setup --codex` 被呼叫時，在 `~/.codex/hooks.json` 中以 merge 方式加入以下 hooks（若原檔不存在或不含 `hooks` 欄位則新建，不覆寫其他已存在的 hooks）：

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

#### Scenario: 首次設定 OpenAI Codex hooks

- **WHEN** `tt setup --codex` 被呼叫，且 `~/.codex/hooks.json` 不存在或不含 `hooks` 欄位
- **THEN** 在 `~/.codex/hooks.json` 中建立 `hooks` 屬性，並在其下加入 `UserPromptSubmit` 與 `Stop` hooks
- **THEN** stdout 輸出 `OpenAI Codex hooks configured in ~/.codex/hooks.json`

### Requirement: Antigravity 與 Codex Stdin Payload 欄位對照

系統 SHALL 於 `tt record` 從 stdin 讀取 JSON payload 時，正確處理 `antigravity` 及 `codex` 的專屬欄位：

1. 當 `--tool` 為 `antigravity` 且 stdin JSON 包含 `conversationId` 欄位時，系統 SHALL 將其映射為 tt 的 `SessionID`；包含 `transcriptPath` 欄位時，SHALL 將其映射為 tt 的 `TranscriptPath`。
2. 當 `--tool` 為 `codex` 時，系統 SHALL 依標準 stdin JSON 格式進行讀取。

#### Scenario: 成功解析 Antigravity 專屬 Stdin Payload

- **WHEN** `tt record prompt --tool antigravity` 被呼叫，且 stdin 輸入為 `{"conversationId": "gemini-session-123", "transcriptPath": "/path/to/transcript.jsonl"}`
- **THEN** 系統建立對應之 session，其 Session ID 為 `gemini-session-123`，且 TranscriptPath 為 `/path/to/transcript.jsonl`
