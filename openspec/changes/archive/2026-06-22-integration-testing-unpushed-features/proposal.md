## Why

為了確保本地已有多個未推送的實作功能（包括分支修復、主動 preempt 搶占、15 分鐘空閒超時以及多工具 log 提取與 fallback）在真實 CLI 呼叫、環境變數隔離和 SQLite 資料庫層面是否正確動作，我們需要設計並實作一個 Go 的端到端整合測試套件。

## What Changes

- 新增一個 Go 整合測試套件，針對真實 CLI 呼叫進行端到端測試，避免 Cobra commands 全域 flag 殘留干擾，並實現完整的環境隔離。
- 模擬並驗證多個工具（Claude Code, Copilot CLI, Google Antigravity）各自不同的 stdin JSON 與 log 檔案的端到端解析與儲存。
- 驗證 Git 分支自動修復、主動 pre-empt 搶占、15 分鐘空閒超時、以及 fallback 預設模型等行為。

## Capabilities

### New Capabilities

- `integration-testing`: 整合測試套件，用於端到端驗證 CLI 的各項行為。

### Modified Capabilities

(none)

## Impact

- 影響程式碼：
  - 新增：
    - `cmd/tt/integration_test.go`
  - 修改：
    - (無)
  - 刪除：
    - (無)

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-22-brainstorm-integration-testing.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
