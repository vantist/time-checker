package transcript

import (
	"math"
	"strings"
)

// TokenEstimator provides token estimation for VS Code Copilot sessions.
type TokenEstimator struct {
	// CharacterToTokenRatio is the default ratio of characters per token.
	CharacterToTokenRatio float64

	// ModelRatios contains model-specific character-to-token ratios.
	ModelRatios map[string]float64
}

// NewTokenEstimator creates a TokenEstimator with default settings.
func NewTokenEstimator() *TokenEstimator {
	return &TokenEstimator{
		CharacterToTokenRatio: 0.25, // ~4 characters per token
		ModelRatios: map[string]float64{
			"gpt-4":    0.25,
			"gpt-4o":   0.25,
			"gpt-5":    0.25,
			"claude":   0.25,
			"o1":       0.25,
		},
	}
}

// EstimateTokensFromText estimates token count from text content.
func (e *TokenEstimator) EstimateTokensFromText(text string, model string) int {
	if len(text) == 0 {
		return 0
	}

	ratio := e.CharacterToTokenRatio
	modelLower := strings.ToLower(model)
	for modelKey, r := range e.ModelRatios {
		if strings.Contains(modelLower, modelKey) {
			ratio = r
			break
		}
	}

	return int(math.Ceil(float64(len(text)) * ratio))
}

// EstimateInputFromOutput estimates input tokens from output tokens based on tool call count.
// Uses different ratios based on session complexity:
//   - >=20 tool calls: 130:1 (heavy agent)
//   - 5-19 tool calls: 50:1 (medium)
//   - <5 tool calls: 10:1 (simple chat)
func (e *TokenEstimator) EstimateInputFromOutput(outputTokens int, toolCallCount int) int {
	var ratio float64
	switch {
	case toolCallCount >= 20:
		ratio = 130
	case toolCallCount >= 5:
		ratio = 50
	default:
		ratio = 10
	}
	return int(math.Round(float64(outputTokens) * ratio))
}

// TokenUsage holds estimated or actual token usage for a session.
type TokenUsage struct {
	InputTokens     int
	OutputTokens    int
	CacheReadTokens int
	Model           string
	IsEstimate      bool
}

// EstimateSessionTokens combines actual token data with estimation to produce a complete picture.
func (e *TokenEstimator) EstimateSessionTokens(
	debugLog *DebugLogResult,
	transcriptContent string,
	toolCallCount int,
	model string,
) TokenUsage {
	// Priority 1: Use actual token counts from debug log
	if debugLog != nil && debugLog.InputTokens > 0 {
		return TokenUsage{
			InputTokens:     debugLog.InputTokens,
			OutputTokens:    debugLog.OutputTokens,
			CacheReadTokens: debugLog.CachedTokens,
			Model:           model,
			IsEstimate:      false,
		}
	}

	// Priority 2: Estimate from transcript content
	outputTokens := e.EstimateTokensFromText(transcriptContent, model)
	inputTokens := e.EstimateInputFromOutput(outputTokens, toolCallCount)

	return TokenUsage{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Model:        model,
		IsEstimate:   true,
	}
}
