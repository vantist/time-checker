package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/user/tt/internal/setup"
)

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.Flags().Bool("claude-code", false, "Set up Claude Code hooks")
	setupCmd.Flags().Bool("copilot", false, "Show Copilot CLI hook setup instructions")
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure AI tool hooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		claudeCode, _ := cmd.Flags().GetBool("claude-code")
		copilot, _ := cmd.Flags().GetBool("copilot")

		if claudeCode {
			if err := setup.SetupClaudeCode(); err != nil {
				return err
			}
			fmt.Println("Claude Code hooks configured in ~/.claude/settings.json")
			return nil
		}

		if copilot {
			fmt.Print(setup.CopilotInstructions)
			return nil
		}

		return cmd.Help()
	},
}
