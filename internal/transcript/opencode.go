package transcript

// OpencodeProvider is a no-op LogProvider for the "opencode" tool.
// opencode tokens come from event flags (via the plugin bridge), so the
// provider is registered only to keep the provider registry complete. None
// of the extraction methods perform token parsing.
type OpencodeProvider struct{}

func (p *OpencodeProvider) ResolvePath(sessionID string, stdinPath string) string {
	return ""
}

func (p *OpencodeProvider) ExtractWindow(path string, fromOffset int, toOffset int) (WindowResult, error) {
	return WindowResult{}, nil
}

func (p *OpencodeProvider) ExtractLastTurn(path string) (WindowResult, error) {
	return WindowResult{}, nil
}

func (p *OpencodeProvider) SupportsSubagents() bool {
	return false
}

func init() {
	Register("opencode", &OpencodeProvider{})
}