# Models Expansion and Robust Suffix Normalization

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

`tt` 時間追蹤器計算 Token 費用時，如果遇到未註冊的 AI 模型，會返回 `nil` (N/A) 成本。隨著使用者整合 `GitHub Copilot CLI`、`Anthropic Claude Code` 與 `Google Antigravity`，許多新模型與版本後綴（如 `-latest`、`-preview`、`-002` 等）在本地日誌中被記錄，導致成本估算錯誤。為了提高計費精準度並防止未來新增變體時失效，需要擴充模型清單並實作穩健的後綴常態化邏輯。

## Decision

擴充 `pricing.go` 的模型費率對照表以涵蓋 2026 年最新主流模型，同時升級 `pricing.normalize` 函數以動態裁切模型版本、預覽或小版本號等後綴，使 `tt` 能夠穩健地對應基礎模型費率。

## Rationale

- **後綴動態裁切（方案 B）優於靜態對照（方案 A）**：因為靜態列舉所有版本變體非常脆弱，一旦官方釋出小改版（例如 `-003`），計費便會破裂。藉由動態裁切後綴，只要基礎模型不變，即使未來推出新版本也能自動相容。
- **涵蓋三大家族主流模型**：完整列入 Anthropic Claude 3/3.5/4/5、OpenAI GPT-4o/GPT-5/o1/o3-mini，以及 Google Gemini 1.5/2.5/3/3.5 等最新計費標準，不漏掉任何主流 AI 工具所產生的日誌。

## Approach

1. **修改常態化邏輯**：在 `internal/pricing/pricing.go` 中，更新 `normalize` 函數：
   - 使用正規表示式裁切常見版本後綴：`-latest`、`-preview`、`-exp`、`-\d{3}`（如 `-001`、`-002`）。
2. **擴充 `table` 費率表**：
   - 納入 Anthropic 3.x/3.5/4/5 費率。
   - 納入 OpenAI GPT-4o, GPT-4o-mini, o1, o1-mini, o3-mini, GPT-5/Codex 系列。
   - 納入 Google Gemini 1.5/2.5/3/3.5 Pro/Flash/Lite 系列。
   - 納入 Copilot 專屬與特定微調模型（`mai-code-1-flash`, `raptor-mini`, `grok-code-fast-1`）。
3. **單元測試防護**：
   - 在 `internal/pricing/pricing_test.go` 中加入對新後綴裁切（例如 `gemini-1.5-pro-002`）以及新增模型計費的測試案例，確保常態化邏輯正確無誤。

## Design Notes

### `pricing.normalize` 實作細節
```go
var dateSuffix = regexp.MustCompile(`-\d{8}$`)
var versionSuffix = regexp.MustCompile(`-(latest|preview|exp|\d{3})$`)

func normalize(model string) string {
	if i := strings.LastIndex(model, "/"); i >= 0 {
		model = model[i+1:]
	}
	model = dateSuffix.ReplaceAllString(model, "")
	model = versionSuffix.ReplaceAllString(model, "")
	return model
}
```

### 擴充之模型費率對照表 (每百萬 Tokens)
| 基礎模型 ID | Input | Output | Cache Read (0.1× 或 OpenAI 50% 折扣) | Cache Write (Anthropic 1.25×) |
|---|---|---|---|---|
| `claude-3-opus` | $15.00 | $75.00 | $1.50 | $18.75 |
| `claude-3-sonnet` | $3.00 | $15.00 | $0.30 | $3.75 |
| `claude-3-haiku` | $0.25 | $1.25 | $0.025 | $0.3125 |
| `claude-3-5-sonnet` | $3.00 | $15.00 | $0.30 | $3.75 |
| `claude-3-5-haiku` | $0.80 | $4.00 | $0.08 | $1.00 |
| `gpt-4o` | $2.50 | $10.00 | $1.25 | $0.00 |
| `gpt-4o-mini` | $0.15 | $0.60 | $0.075 | $0.00 |
| `o1` | $15.00 | $60.00 | $7.50 | $0.00 |
| `o1-mini` | $3.00 | $12.00 | $1.50 | $0.00 |
| `o3-mini` | $1.10 | $4.40 | $0.55 | $0.00 |
| `gpt-5.3-codex` | $1.75 | $14.00 | $0.875 | $0.00 |
| `gpt-5.4-codex` | $2.50 | $15.00 | $1.25 | $0.00 |
| `gpt-5.5-codex` | $5.00 | $30.00 | $2.50 | $0.00 |
| `gpt-5.4-mini` | $0.75 | $3.00 | $0.375 | $0.00 |
| `gpt-5.5` | $5.00 | $30.00 | $2.50 | $0.00 |
| `mai-code-1-flash` | $0.75 | $4.50 | $0.075 | $0.00 |
| `raptor-mini` / `raptor_mini` | $0.25 | $2.00 | $0.025 | $0.00 |
| `grok-code-fast-1` | $1.00 | $2.00 | $0.10 | $0.00 |
| `gemini-1.5-pro` | $1.25 | $5.00 | $0.125 | $0.00 |
| `gemini-1.5-flash` | $0.075 | $0.30 | $0.0075 | $0.00 |
| `gemini-2.5-pro` | $1.25 | $10.00 | $0.125 | $0.00 |
| `gemini-2.5-flash` | $0.30 | $2.50 | $0.03 | $0.00 |
| `gemini-2.5-flash-lite` | $0.10 | $0.40 | $0.01 | $0.00 |
| `gemini-3-flash` | $0.50 | $3.00 | $0.05 | $0.00 |
| `gemini-3.1-pro` | $2.00 | $12.00 | $0.20 | $0.00 |
| `gemini-3.1-flash-lite` | $0.25 | $1.50 | $0.025 | $0.00 |
| `gemini-3.5-flash` | $1.50 | $9.00 | $0.15 | $0.00 |

## Insights to Capture

- `design.md`: 說明新增的版本與後綴裁切（normalize）邏輯，以及擴充費率表之原因與規格。
- `specs/model-cost-tracking/spec.md`: 增列新模型定價與後綴常態化驗證情境（Scenarios）。
- `proposal.md`: 加入新模型支援與常態化實作範疇。
- `tasks.md`: 新增常態化邏輯實作、費率表擴充、單元測試編寫與驗證任務。

## Open Questions

（無，所有設計與費率細節已在 Brainstorm 期間與使用者確認收斂）
