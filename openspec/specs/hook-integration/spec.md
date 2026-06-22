# hook-integration Specification

## Purpose
TBD - created by archiving change ai-tool-time-tracker. Update Purpose after archive.
## Requirements
### Requirement: Claude Code Hook 設定

系統 SHALL 在 `tt setup --claude-code` 被呼叫時，在 `~/.claude/settings.json` 中加入以下 hooks（不覆蓋現有 hooks，以 merge 方式加入）：

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "tt record prompt --session $CLAUDE_SESSION_ID --project $CLAUDE_PROJECT_PATH --tool claude-code --model $CLAUDE_MODEL"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "tt record response --session $CLAUDE_SESSION_ID --tokens \"$CLAUDE_USAGE_JSON\""
          }
        ]
      }
    ]
  }
}
```

#### Scenario: 首次設定 Claude Code hooks

- **WHEN** `tt setup --claude-code` 被呼叫，且 `~/.claude/settings.json` 不存在或不含 `hooks` 欄位
- **THEN** 在 `~/.claude/settings.json` 中加入 `UserPromptSubmit` 與 `Stop` hooks
- **THEN** stdout 輸出 `Claude Code hooks configured in ~/.claude/settings.json`

#### Scenario: 設定時保留現有 hooks

- **WHEN** `~/.claude/settings.json` 已存在其他 hooks（例如 caveman mode hook）
- **THEN** 現有 hooks 不被覆蓋或刪除，tt hooks 以追加方式加入

### Requirement: Claude Code Hook 事件欄位對照

系統 SHALL 正確解析 Claude Code hook 呼叫時的環境變數或 stdin payload：

| 事件 | 資料來源 | 欄位 |
|------|----------|------|
| `UserPromptSubmit` | 環境變數 | `CLAUDE_SESSION_ID`, `CLAUDE_PROJECT_PATH`, `CLAUDE_MODEL` |
| `Stop` | 環境變數 | `CLAUDE_SESSION_ID`, `CLAUDE_USAGE_JSON`（JSON 字串） |

**注意**：實際欄位名稱需在實作時確認 Claude Code hook 文件，若與上述不符，以實際文件為準。

#### Scenario: Stop hook 帶有完整 token 資訊

- **WHEN** Claude Code `Stop` hook 觸發，`CLAUDE_USAGE_JSON = '{"input_tokens":5000,"output_tokens":800,"cache_read_tokens":3000,"cache_creation_tokens":0}'`
- **THEN** `tt record response` 正確解析並寫入所有 token 欄位

#### Scenario: Stop hook 不帶 token 資訊

- **WHEN** Claude Code `Stop` hook 觸發，`CLAUDE_USAGE_JSON` 為空或不存在
- **THEN** `tt record response` 寫入 `response_at` 並更新 `ended_at`，token 欄位寫入 NULL

### Requirement: Copilot CLI Hook 設定說明

系統 SHALL 透過 `tt setup --copilot` 輸出 Copilot CLI hook 設定的指引（不自動寫入，因 Copilot CLI hook 路徑因版本而異）。

#### Scenario: 輸出 Copilot CLI 設定指引

- **WHEN** `tt setup --copilot` 被呼叫
- **THEN** stdout 輸出包含 Copilot CLI hooks 目錄路徑、事件名稱（`userPromptSubmitted`, `agentStop`）、以及對應的 `tt record` 命令

### Requirement: Copilot CLI Hook 事件欄位對照

系統 SHALL 支援 Copilot CLI hook 呼叫格式，事件欄位對照如下：

| Copilot CLI 事件 | 對應 Claude Code 事件 | 備註 |
|-----------------|----------------------|------|
| `userPromptSubmitted` | `UserPromptSubmit` | 觸發 `tt record prompt` |
| `agentStop` | `Stop` | 觸發 `tt record response` |

#### Scenario: Copilot CLI agentStop 無 token 資料時不報錯

- **WHEN** Copilot CLI `agentStop` hook 觸發，payload 不包含 token 資訊
- **THEN** `tt record response --tokens '{}'` 被呼叫，token 欄位寫入 NULL
- **THEN** exit code 0

### Requirement: Hook 參數解析安全 Fallback 與預設 Model 載入

當 hook 執行並解析 Prompt 輸入時，系統 SHALL 依據以下邏輯進行參數的 Fallback 與解析：
1. **專案路徑 Fallback**：若 CLI 參數與 stdin payload 均未提供 `project` 路徑，系統 SHALL 使用當前行程之工作目錄 (`os.Getwd()`) 作為專案路徑。
2. **Antigravity 預設 Model 載入**：若整合工具為 `"antigravity"` 且 model 為空字串時，系統 SHALL 主動由 `settings.json`（透過 `GetAntigravityModel`）載入並填入預設的模型名稱。

#### Scenario: 專案路徑 Fallback 至工作目錄
- **WHEN** 呼叫 `tt record prompt` 且未提供 `--project` 且 stdin 無 `cwd` 資訊
- **THEN** 系統呼叫 `os.Getwd()` 取得當前工作目錄並填入 `sessions.project`

#### Scenario: Antigravity 工具無模型名稱時主動載入設定檔預設值
- **WHEN** 呼叫 `tt record prompt --tool antigravity` 且未提供 `--model`，且 `settings.json` 已配置預設模型為 `gemini-2.5-pro`
- **THEN** 系統解析 model 為 `gemini-2.5-pro`

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

