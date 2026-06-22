## ADDED Requirements

### Requirement: SetupCopilot 為 idempotent 操作

`tt setup --copilot` 的 `SetupCopilot()` 函數 SHALL 為 idempotent：多次執行的最終結果與執行一次相同。每個由 tt 所有的 hook 條目 SHALL 帶有 `"_owner": "tt"` 標記欄位。merge 邏輯 SHALL 先移除所有 `_owner == "tt"` 的舊條目，再插入新版本，確保不產生重複條目且更新後舊版本被移除。非 tt 所有的 hook 條目 SHALL NOT 受影響。
Copilot CLI 的 hooks 設定檔路徑 SHALL 為 `~/.copilot/hooks/tt.json`。
其 JSON 結構格式 SHALL 如下：
```json
{
  "version": 1,
  "hooks": {
    "userPromptSubmitted": [
      {
        "_owner": "tt",
        "type": "command",
        "command": "tt record prompt --tool copilot-cli"
      }
    ],
    "agentStop": [
      {
        "_owner": "tt",
        "type": "command",
        "command": "tt record response --tool copilot-cli"
      }
    ]
  }
}
```

#### Scenario: 重複執行不產生重複條目

- **WHEN** `SetupCopilot()` 在同一個 `tt.json` 上執行兩次
- **THEN** 每個 event（`userPromptSubmitted`、`agentStop`）下各自只有一個 tt hook 條目，不出現重複

#### Scenario: 更新 hook 內容後重新 setup 取代舊版本

- **WHEN** `ttHooks` 的 command 字串更新後，`SetupCopilot()` 在已含舊版本 hook 的 `tt.json` 上執行
- **THEN** 舊版本 hook 條目被移除，新版本條目被插入，event 下只存在新版本

#### Scenario: 不影響使用者自有 hook

- **WHEN** `tt.json` 的某個 event 已存在 `_owner` 不為 `"tt"` 的 hook 條目，`SetupCopilot()` 執行
- **THEN** 該使用者自有條目仍保留在結果中，未被移除或修改

#### Scenario: 首次安裝

- **WHEN** `tt.json` 檔案不存在，或 hooks 為空，`SetupCopilot()` 執行
- **THEN** 系統建立/合併 `tt.json` 檔案，`version` 欄位設為 1，並在 `userPromptSubmitted` 與 `agentStop` 各新增一個帶 `"_owner": "tt"` 的 hook 條目
