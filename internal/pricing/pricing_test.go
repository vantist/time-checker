package pricing_test

import (
	"testing"

	"github.com/user/tt/internal/pricing"
)

// Task 4.1: Scenario 1 from cost-estimation spec
// model=claude-sonnet-4-6, input=1000, output=200, cache_read=500, cache_creation=0
// cost = (1000/1e6)*3.00 + (200/1e6)*15.00 + (500/1e6)*0.30 + 0
//      = 0.003 + 0.003 + 0.00015 = 0.00615
func TestCalculateSonnet(t *testing.T) {
	got := pricing.Calculate("claude-sonnet-4-6", 1000, 200, 500, 0)
	if got == nil {
		t.Fatal("expected non-nil cost")
	}
	const want = 0.00615
	if *got < want-0.000001 || *got > want+0.000001 {
		t.Errorf("cost = %f, want ~%f", *got, want)
	}
}

// Task 4.3: unknown model returns nil
func TestCalculateUnknownModelNil(t *testing.T) {
	got := pricing.Calculate("gpt-5-unknown", 1000, 200, 0, 0)
	if got != nil {
		t.Errorf("expected nil for unknown model, got %v", *got)
	}
}
