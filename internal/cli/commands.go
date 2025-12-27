package cli

import "github.com/spf13/cobra"

// Commands returns all CLI subcommands for the stacktower application.
// This centralizes command registration and makes it harder to forget
// to add new commands to the CLI.
func Commands() []*cobra.Command {
	return []*cobra.Command{
		// Core pipeline commands
		NewParseCmd(),
		NewLayoutCmd(),
		NewVisualizeCmd(),
		NewRenderCmd(),

		// Utility commands
		NewCacheCmd(),
		NewPQTreeCmd(),

		// GitHub integration
		NewGitHubCmd(),
	}
}
