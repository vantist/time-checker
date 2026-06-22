## ADDED Requirements

### Requirement: SetupAntigravity 為 idempotent 操作

`tt setup --antigravity` 的 `SetupAntigravity()` 函數 SHALL 為 idempotent：多次執行的最終結果與執行一次相同。每個由 tt 所有的 hook 條目 SHALL 帶有 `"_owner": "tt"` 標記欄位。merge 邏輯 SHALL 先移除所有 `_owner == "tt"` 的舊條目，再插入新版本，確保不產生重複條目且更新後舊版本被移除。非 tt 所有的 hook 條目 SHALL NOT 受影響。

#### Scenario: Antigravity hooks 重複執行不產生重複條目

- **WHEN** `SetupAntigravity()` 在同一個 `hooks.json` 上執行兩次
- **THEN** 每個 event（`PreInvocation`、`Stop`）下各自只有一個 tt hook 條目，不出現重複

#### Scenario: Antigravity hooks 首次安裝

- **WHEN** `hooks.json` 不存在或為空，`SetupAntigravity()` 執行
- **THEN** `PreInvocation` 與 `Stop` 各新增一個帶 `"_owner": "tt"` 的 hook 條目

### Requirement: SetupCodex 為 idempotent 操作

`tt setup --codex` 的 `SetupCodex()` 函數 SHALL 為 idempotent：多次執行的最終結果與執行一次相同。每個由 tt 所有的 hook 條目 SHALL 帶有 `"_owner": "tt"` 標記欄位。merge 邏輯 SHALL 先移除所有 `_owner == "tt"` 的舊條目，再插入新版本，確保不產生重複條目且更新後舊版本被移除。非 tt 所有的 hook 條目 SHALL NOT 受影響。

#### Scenario: Codex hooks 重複執行不產生重複條目

- **WHEN** `SetupCodex()` 在同一個 `hooks.json` 上執行兩次
- **THEN** 每個 event（`UserPromptSubmit` , `Stop`）下各自只有一個 tt hook 條目，不出現重複

#### Scenario: Codex hooks 首次安裝

- **WHEN** `hooks.json` 不存在或為空，`SetupCodex()` 執行
- **THEN** `UserPromptSubmit` 與 `Stop` 各新增一個帶 `"_owner": "tt"` 的 hook 條目
