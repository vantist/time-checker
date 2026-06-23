package transcript

import (
	"testing"
)

func TestTokenEstimator_EstimateTokensFromText(t *testing.T) {
	est := NewTokenEstimator()

	tests := []struct {
		name     string
		text     string
		model    string
		expected int
	}{
		{
			name:     "empty text",
			text:     "",
			model:    "gpt-4o",
			expected: 0,
		},
		{
			name:     "short text",
			text:     "hello world",
			model:    "gpt-4o",
			expected: 3, // 11 chars * 0.25 = 2.75, ceil = 3
		},
		{
			name:     "longer text",
			text:     "This is a test message with some content",
			model:    "gpt-4o",
			expected: 10, // 41 chars * 0.25 = 10.25, ceil = 11
		},
		{
			name:     "unknown model uses default ratio",
			text:     "hello",
			model:    "unknown-model",
			expected: 2, // 5 chars * 0.25 = 1.25, ceil = 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := est.EstimateTokensFromText(tt.text, tt.model)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestTokenEstimator_EstimateInputFromOutput(t *testing.T) {
	est := NewTokenEstimator()

	tests := []struct {
		name          string
		outputTokens  int
		toolCallCount int
		expectedRatio float64
	}{
		{
			name:          "simple chat",
			outputTokens:  100,
			toolCallCount: 2,
			expectedRatio: 10,
		},
		{
			name:          "medium agent",
			outputTokens:  100,
			toolCallCount: 10,
			expectedRatio: 50,
		},
		{
			name:          "heavy agent",
			outputTokens:  100,
			toolCallCount: 25,
			expectedRatio: 130,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := est.EstimateInputFromOutput(tt.outputTokens, tt.toolCallCount)
			expected := int(tt.expectedRatio * float64(tt.outputTokens))
			if result != expected {
				t.Errorf("expected %d, got %d", expected, result)
			}
		})
	}
}

func TestEstimateSessionTokens_WithDebugLog(t *testing.T) {
	est := NewTokenEstimator()

	debugLog := &DebugLogResult{
		InputTokens:  500,
		OutputTokens: 200,
		CachedTokens: 100,
	}

	result := est.EstimateSessionTokens(debugLog, "some content", 5, "gpt-4o")

	if result.IsEstimate {
		t.Error("expected actual tokens, not estimate")
	}
	if result.InputTokens != 500 {
		t.Errorf("expected 500 input tokens, got %d", result.InputTokens)
	}
	if result.OutputTokens != 200 {
		t.Errorf("expected 200 output tokens, got %d", result.OutputTokens)
	}
	if result.CacheReadTokens != 100 {
		t.Errorf("expected 100 cache read tokens, got %d", result.CacheReadTokens)
	}
}

func TestEstimateSessionTokens_WithoutDebugLog(t *testing.T) {
	est := NewTokenEstimator()

	result := est.EstimateSessionTokens(nil, "hello world", 2, "gpt-4o")

	if !result.IsEstimate {
		t.Error("expected estimated tokens")
	}
	if result.InputTokens == 0 {
		t.Error("expected non-zero input tokens")
	}
	if result.OutputTokens == 0 {
		t.Error("expected non-zero output tokens")
	}
}

func TestNewTokenEstimator_Defaults(t *testing.T) {
	est := NewTokenEstimator()

	if est.CharacterToTokenRatio != 0.25 {
		t.Errorf("expected default ratio 0.25, got %f", est.CharacterToTokenRatio)
	}
	if len(est.ModelRatios) == 0 {
		t.Error("expected model ratios to be populated")
	}
}
