package transcript

import (
	"testing"
)

// Task 1.1: registering "opencode" provider for registry completeness —
// it must not perform any token extraction (opencode tokens come from events).
func TestOpenCodeProviderRegistered(t *testing.T) {
	p, ok := GetProvider("opencode")
	if !ok {
		t.Fatal("expected opencode provider to be registered")
	}
	if p == nil {
		t.Fatal("opencode provider is nil")
	}
	if p.SupportsSubagents() {
		t.Error("opencode provider should not declare subagent support (tokens come from events)")
	}
}

// TestOpenCodeProvider_DoesNotExtract: extraction calls always return empty results
// (opencode tokens come from event flags, not transcript parsing).
func TestOpenCodeProvider_DoesNotExtract(t *testing.T) {
	p, ok := GetProvider("opencode")
	if !ok {
		t.Fatal("expected opencode provider to be registered")
	}

	res, err := p.ExtractWindow("/nonexistent/path.jsonl", 0, -1)
	if err != nil {
		t.Errorf("ExtractWindow should not error on opencode provider, got: %v", err)
	}
	if res.InputTokens() != 0 || res.OutputTokens() != 0 {
		t.Errorf("ExtractWindow should return empty result, got in=%d out=%d",
			res.InputTokens(), res.OutputTokens())
	}

	res2, err := p.ExtractLastTurn("/nonexistent/path.jsonl")
	if err != nil {
		t.Errorf("ExtractLastTurn should not error on opencode provider, got: %v", err)
	}
	if res2.InputTokens() != 0 || res2.OutputTokens() != 0 {
		t.Errorf("ExtractLastTurn should return empty result, got in=%d out=%d",
			res2.InputTokens(), res2.OutputTokens())
	}
}