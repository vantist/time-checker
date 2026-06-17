package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/user/tt/internal/db"
	"github.com/user/tt/internal/recorder"
)

func init() {
	rootCmd.AddCommand(recordCmd)
	recordCmd.AddCommand(recordPromptCmd, recordResponseCmd)

	recordPromptCmd.Flags().String("session", "", "session ID (overrides stdin)")
	recordPromptCmd.Flags().String("project", "", "project path (overrides stdin)")
	recordPromptCmd.Flags().String("tool", "claude-code", "tool name")
	recordPromptCmd.Flags().String("model", "", "model name (overrides stdin)")

	recordResponseCmd.Flags().String("session", "", "session ID (overrides stdin)")
	recordResponseCmd.Flags().String("tokens", "", "tokens JSON string (overrides stdin)")
	recordResponseCmd.Flags().String("tool", "claude-code", "tool name")
}

var recordCmd = &cobra.Command{
	Use:   "record",
	Short: "Record AI tool events (called by hooks)",
}

var recordPromptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Record a user prompt event",
	RunE: func(cmd *cobra.Command, args []string) error {
		input, err := resolvePromptInput(cmd)
		if err != nil {
			return err
		}

		conn, err := db.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "tt: db open error: %v\n", err)
			return nil
		}
		defer conn.Close()

		return recorder.RecordPromptSilent(conn, input)
	},
}

var recordResponseCmd = &cobra.Command{
	Use:   "response",
	Short: "Record a response/stop event",
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID, tokensJSON, err := resolveResponseInput(cmd)
		if err != nil {
			return err
		}

		conn, err := db.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "tt: db open error: %v\n", err)
			return nil
		}
		defer conn.Close()

		return recorder.RecordResponseSilent(conn, sessionID, tokensJSON)
	},
}

// hookPayload covers both Claude Code and Copilot CLI stdin formats.
type hookPayload struct {
	// Claude Code fields
	SessionID      string `json:"session_id"`
	Cwd            string `json:"cwd"`
	Model          string `json:"model"`
	TranscriptPath string `json:"transcript_path"`
	// Copilot CLI fields
	CopilotSessionID string `json:"sessionId"`
}

func readStdinJSON() (*hookPayload, error) {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil, nil // interactive terminal, no stdin
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil || len(data) == 0 {
		return nil, err
	}
	var p hookPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, nil // malformed, ignore
	}
	// normalise Copilot sessionId → session_id
	if p.SessionID == "" && p.CopilotSessionID != "" {
		p.SessionID = p.CopilotSessionID
	}
	return &p, nil
}

func resolvePromptInput(cmd *cobra.Command) (recorder.PromptInput, error) {
	stdin, _ := readStdinJSON()

	sessionID, _ := cmd.Flags().GetString("session")
	project, _ := cmd.Flags().GetString("project")
	tool, _ := cmd.Flags().GetString("tool")
	model, _ := cmd.Flags().GetString("model")

	if stdin != nil {
		if sessionID == "" {
			sessionID = stdin.SessionID
		}
		if project == "" {
			project = stdin.Cwd
		}
		if model == "" {
			model = stdin.Model
		}
	}

	return recorder.PromptInput{
		SessionID: sessionID,
		Project:   project,
		Tool:      tool,
		Model:     model,
	}, nil
}

func resolveResponseInput(cmd *cobra.Command) (sessionID, tokensJSON string, err error) {
	stdin, _ := readStdinJSON()

	sessionID, _ = cmd.Flags().GetString("session")
	tokensJSON, _ = cmd.Flags().GetString("tokens")

	if stdin != nil {
		if sessionID == "" {
			sessionID = stdin.SessionID
		}
		if tokensJSON == "" && stdin.TranscriptPath != "" {
			tokensJSON = extractTokensFromTranscript(stdin.TranscriptPath)
		}
	}
	return sessionID, tokensJSON, nil
}

// extractTokensFromTranscript reads the transcript JSONL and returns the usage
// from the last assistant message as a flat JSON string.
func extractTokensFromTranscript(path string) string {
	// expand ~ if present
	if len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	type usageFields struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	}
	type transcriptEntry struct {
		Type    string `json:"type"`
		Message struct {
			Usage usageFields `json:"usage"`
		} `json:"message"`
	}

	var last *usageFields
	dec := json.NewDecoder(f)
	for dec.More() {
		var entry transcriptEntry
		if err := dec.Decode(&entry); err != nil {
			continue
		}
		if entry.Type == "assistant" {
			u := entry.Message.Usage
			last = &u
		}
	}
	if last == nil {
		return ""
	}

	out, err := json.Marshal(map[string]int{
		"input_tokens":        last.InputTokens,
		"output_tokens":       last.OutputTokens,
		"cache_read_tokens":   last.CacheReadInputTokens,
		"cache_creation_tokens": last.CacheCreationInputTokens,
	})
	if err != nil {
		return ""
	}
	return string(out)
}
