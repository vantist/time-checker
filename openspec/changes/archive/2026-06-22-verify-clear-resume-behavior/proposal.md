## Why

使用者提出疑問，希望確認系統中 `/clear`（清除歷史紀錄）與 `--resume`（重新關聯進程/Resume 對話）在整個架構中的運作行為是否有被正確實作，以及是否會產生資料衝突或時間統計上的異常。本變更旨在透過架構審查、補強單元測試，驗證並對齊此二行為的設計與運作流程。

## What Changes

- 驗證 `/clear` 之後繼續對話的 Session 級別對齊與 Turn 級別隔離。
- 驗證進程重新開啟並指定 `--resume` 時的 Session 與 Process Key 復原行為。
- 驗證 Antigravity 整合特性（單一 Turn 去重機制與 Clear Race 補救機制）。
- 在 `stable-session-key` 規格書中補強並落實 `/clear` 與 `--resume` 的行為定義，確保其作為系統開發與測試的依據。

## Capabilities

### New Capabilities
- (none)

### Modified Capabilities
- `stable-session-key`: 補充 `/clear` 與 `--resume` 在 Session/Turn 級別與 Reconcile 時的具體行為規範與驗證場景。

## Impact

- Affected specs:
  - `openspec/specs/stable-session-key/spec.md`
- Affected code:
  - Modified:
    - `internal/db/session_test.go`
    - `internal/transcript/extract_test.go`
    - `internal/recorder/recorder_test.go`

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-21-brainstorm-clear-resume-validation.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
