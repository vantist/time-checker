package pricing_test

import (
	"testing"

	"github.com/user/tt/internal/pricing"
	"github.com/user/tt/internal/transcript"
)

func assertCost(t *testing.T, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Fatal("expected non-nil cost")
	}
	if *got < want-1e-6 || *got > want+1e-6 {
		t.Errorf("cost = %f, want ~%f", *got, want)
	}
}

// Task 4.1: Scenario 1 from cost-estimation spec
// model=claude-sonnet-4-6, input=1000, output=200, cache_read=500, cache_creation=0
// cost = (1000/1e6)*3.00 + (200/1e6)*15.00 + (500/1e6)*0.30 + 0
//      = 0.003 + 0.003 + 0.00015 = 0.00615
func TestCalculateSonnet(t *testing.T) {
	got := pricing.Calculate("claude-sonnet-4-6", 1000, 200, 500, 0, 0, 0)
	assertCost(t, got, 0.00615)
}

// Task 4.3: unknown model returns nil
func TestCalculateUnknownModelNil(t *testing.T) {
	got := pricing.Calculate("gpt-5-unknown", 1000, 200, 0, 0, 0, 0)
	if got != nil {
		t.Errorf("expected nil for unknown model, got %v", *got)
	}
}

// Task 16.1: vertex_ai prefix model correctly looks up pricing
func TestCalculateVertexAIPrefix(t *testing.T) {
	got := pricing.Calculate("vertex_ai/claude-sonnet-4-6", 1_000_000, 0, 0, 0, 0, 0)
	assertCost(t, got, 3.00)
}

// Task 16.2: claude-haiku-4-5 (no prefix) priced at $1.00/MTok
func TestCalculateHaiku45(t *testing.T) {
	got := pricing.Calculate("claude-haiku-4-5", 1_000_000, 0, 0, 0, 0, 0)
	assertCost(t, got, 1.00)
}

// Task 16.3: claude-opus-4-8 priced at $5.00/MTok (not old $15.00)
func TestCalculateOpus48NewPricing(t *testing.T) {
	got := pricing.Calculate("claude-opus-4-8", 1_000_000, 0, 0, 0, 0, 0)
	assertCost(t, got, 5.00)
}

// Date-suffix stripping: claude-haiku-4-5-20251001 should resolve to claude-haiku-4-5
func TestCalculateDateSuffix(t *testing.T) {
	got := pricing.Calculate("claude-haiku-4-5-20251001", 1_000_000, 0, 0, 0, 0, 0)
	assertCost(t, got, 1.00)
}

// Task 16.4: unknown model after normalize returns nil
func TestCalculateUnknownAfterNormalize(t *testing.T) {
	got := pricing.Calculate("vertex_ai/gpt-5-unknown", 1000, 0, 0, 0, 0, 0)
	if got != nil {
		t.Errorf("expected nil for unknown model, got %v", *got)
	}
}

// TestCalculate_Cache5m1h: 5m and 1h cache creation priced at different rates.
// claude-sonnet-4-6: input=$3/MTok, output=$15/MTok, cacheRead=$0.30/MTok, cacheCreation=$3.75/MTok
// 5m tokens: 1000, 1h tokens: 2000 — both use cacheCreation rate ($3.75/MTok)
// cost = (1000+2000)/1e6 * 3.75 = 0.01125
func TestCalculate_Cache5m1h(t *testing.T) {
	got := pricing.Calculate("claude-sonnet-4-6", 0, 0, 0, 0, 1000, 2000)
	assertCost(t, got, 0.01125)
}

func TestCalculateForUsage(t *testing.T) {
	u := transcript.ModelUsage{
		Model:               "claude-sonnet-4-6",
		InputTokens:         1000,
		OutputTokens:        200,
		CacheReadTokens:     500,
		CacheCreationTokens: 0,
		CacheCreation5m:     1000,
		CacheCreation1h:     2000,
	}
	got := pricing.CalculateForUsage(u)
	assertCost(t, got, 0.01740)
}

func TestCalculateGpt54(t *testing.T) {
	got := pricing.Calculate("gpt-5.4", 1_000_000, 0, 0, 0, 0, 0)
	assertCost(t, got, 5.00)
}

func TestCalculateGpt5Mini(t *testing.T) {
	got := pricing.Calculate("gpt-5-mini", 1_000_000, 0, 0, 0, 0, 0)
	assertCost(t, got, 0.15)
}

func TestCalculateSuffixNormalization(t *testing.T) {
	// Test normalization with existing base models in table
	assertCost(t, pricing.Calculate("claude-haiku-4-5-latest", 1_000_000, 0, 0, 0, 0, 0), 1.00)
	assertCost(t, pricing.Calculate("claude-haiku-4-5-preview", 1_000_000, 0, 0, 0, 0, 0), 1.00)
	assertCost(t, pricing.Calculate("claude-haiku-4-5-exp", 1_000_000, 0, 0, 0, 0, 0), 1.00)
	assertCost(t, pricing.Calculate("claude-haiku-4-5-002", 1_000_000, 0, 0, 0, 0, 0), 1.00)

	// Test normalization with 2026 models (will fail because suffix normalize is missing AND base models are not in table yet)
	assertCost(t, pricing.Calculate("gemini-1.5-pro-002", 1_000_000, 0, 0, 0, 0, 0), 1.25)
	assertCost(t, pricing.Calculate("claude-3-5-sonnet-latest", 1_000_000, 0, 0, 0, 0, 0), 3.00)
	assertCost(t, pricing.Calculate("gpt-4o-preview", 1_000_000, 0, 0, 0, 0, 0), 2.50)
}

func TestCalculate2026Models(t *testing.T) {
	// gemini-3.5-flash: input=$1.50/MTok
	assertCost(t, pricing.Calculate("gemini-3.5-flash", 1_000_000, 0, 0, 0, 0, 0), 1.50)

	// claude-3-5-sonnet: input=$3.00/MTok, cache write=$3.75/MTok
	// input 1,000,000 tokens, cache write 1,000,000 tokens, cost = 3.00 + 3.75 = 6.75
	assertCost(t, pricing.Calculate("claude-3-5-sonnet", 1_000_000, 0, 0, 1_000_000, 0, 0), 6.75)

	// o1: input=$15.00/MTok, output=$60.00/MTok
	// input 1,000,000 tokens, output 500,000 tokens, cost = 15.00 + 30.00 = 45.00
	assertCost(t, pricing.Calculate("o1", 1_000_000, 500_000, 0, 0, 0, 0), 45.00)

	// gpt-4o: input=$2.50/MTok, cache read=$1.25/MTok
	// input 1,000,000 tokens, cache read 2,000,000 tokens, cost = 2.50 + 2.50 = 5.00
	assertCost(t, pricing.Calculate("gpt-4o", 1_000_000, 0, 2_000_000, 0, 0, 0), 5.00)

	// grok-code-fast-1: input=$1.00/MTok, output=$2.00/MTok
	assertCost(t, pricing.Calculate("grok-code-fast-1", 1_000_000, 1_000_000, 0, 0, 0, 0), 3.00)

	// mai-code-1-flash: input=$0.75/MTok, output=$4.50/MTok
	assertCost(t, pricing.Calculate("mai-code-1-flash", 2_000_000, 1_000_000, 0, 0, 0, 0), 6.00)

	// raptor-mini: input=$0.25/MTok, output=$2.00/MTok
	assertCost(t, pricing.Calculate("raptor-mini", 4_000_000, 1_000_000, 0, 0, 0, 0), 3.00)

	// gemini-2.5-flash-lite: input=$0.10/MTok, output=$0.40/MTok
	assertCost(t, pricing.Calculate("gemini-2.5-flash-lite", 10_000_000, 5_000_000, 0, 0, 0, 0), 3.00)

	// claude-fable-5: input=$10.00/MTok, cache read=$1.00/MTok
	assertCost(t, pricing.Calculate("claude-fable-5", 1_000_000, 0, 1_000_000, 0, 0, 0), 11.00)
}


