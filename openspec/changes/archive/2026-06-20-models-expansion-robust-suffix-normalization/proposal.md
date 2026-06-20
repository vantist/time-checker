## Why

`tt` 時間追蹤器計算 Token 費用時，如果遇到未註冊的 AI 模型，會返回 `nil` (N/A) 成本。隨著使用者整合 `GitHub Copilot CLI`、`Anthropic Claude Code` 與 `Google Antigravity`，許多新模型與版本後綴（如 `-latest`、`-preview`、`-exp`、`-002` 等）在本地日誌中被記錄，導致成本估算錯誤。為了提高計費精準度並防止未來新增變體時失效，需要擴充模型清單並實作穩健的後綴常態化邏輯。

## What Changes

- **後綴常態化邏輯升級**：在 `internal/pricing/pricing.go` 的 `normalize` 函數中，除了去除最後一個 `/` 之前的所有字元與移除 `-\d{8}$` 之外，還會以正規表示式裁切常見版本後綴：`-latest`、`-preview`、`-exp`、`-\d{3}`（如 `-001`、`-002` 等）。
- **費率表 (Pricing Table) 擴充**：擴充 `table` 費率表以涵蓋 2026 年最新主流模型，包括 Anthropic Claude 3/3.5/4/5、OpenAI GPT-4o/GPT-5/o1/o3-mini、Google Gemini 1.5/2.5/3/3.5 Pro/Flash/Lite 系列，以及其他特定模型費率（如 `mai-code-1-flash`、`raptor-mini`、`grok-code-fast-1`）。
- **完整單元測試**：在 `internal/pricing/pricing_test.go` 中實作測試案例，驗證新增的模型費率計算與後綴裁切邏輯正確無誤。

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `model-cost-tracking`: 擴充費率表以涵蓋 2026 年最新主流模型，並升級 `pricing.normalize` 邏輯以動態裁切模型版本、預覽或小版本號等後綴，使 `tt` 能夠穩健地對應基礎模型費率。

## Impact

- Affected specs: `model-cost-tracking`
- Affected code:
  - Modified:
    - `internal/pricing/pricing.go`
    - `internal/pricing/pricing_test.go`

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-20-brainstorm-models-expansion.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
