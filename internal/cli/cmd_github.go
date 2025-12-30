package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/internal/cli/term"
	"github.com/matzehuels/stacktower/pkg/integrations/github"
)

// NewGitHubCmd creates the github command with subcommands.
func NewGitHubCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "github",
		Short: "GitHub integration commands",
		Long: `Authenticate with GitHub and interact with your repositories.

Use the device flow to authenticate without needing a web browser callback.
Your session is stored in ~/.stacktower/sessions/`,
	}

	cmd.AddCommand(newGitHubLoginCmd())
	cmd.AddCommand(newGitHubLogoutCmd())
	cmd.AddCommand(newGitHubWhoamiCmd())

	return cmd
}

// newGitHubLoginCmd creates the login subcommand.
func newGitHubLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with GitHub using device flow",
		Long: `Start the GitHub device authorization flow.

You'll be given a code to enter at https://github.com/login/device.
Once authorized, your session will be saved locally for future commands.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			// Check if already logged in
			if existing, _ := LoadGitHubSession(ctx); existing != nil {
				term.PrintInfo("Already logged in as @%s", existing.User.Login)
				term.PrintDetail("Run 'stacktower github logout' first to re-authenticate")
				return nil
			}

			_, err := runGitHubLogin(ctx)
			return err
		},
	}

	return cmd
}

// newGitHubLogoutCmd creates the logout subcommand.
func newGitHubLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored GitHub credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := deleteGitHubSession(cmd.Context()); err != nil {
				return err
			}
			term.PrintSuccess("Logged out")
			return nil
		},
	}
}

// newGitHubWhoamiCmd creates the whoami subcommand.
func newGitHubWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the currently authenticated GitHub user",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := LoadGitHubSession(cmd.Context())
			if err != nil {
				return err
			}

			// Verify token still works by fetching fresh user info
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			spinner := term.NewSpinner("Verifying session...")
			spinner.Start()

			client := github.NewContentClient(sess.AccessToken)
			user, err := client.FetchUser(ctx)
			if err != nil {
				spinner.StopWithError("Session invalid")
				return fmt.Errorf("session may be invalid: %w", err)
			}
			spinner.Stop()

			// Show session info
			summary := term.NewSuccessSummary("GitHub Session")
			summary.AddKeyValue("Username", "@"+user.Login)
			if user.Name != "" {
				summary.AddKeyValue("Name", user.Name)
			}
			if user.Email != "" {
				summary.AddKeyValue("Email", user.Email)
			}
			summary.AddKeyValue("Logged in", sess.CreatedAt.Format("Jan 2, 2006"))
			summary.AddKeyValue("Expires", sess.ExpiresAt.Format("Jan 2, 2006"))
			summary.Print()

			return nil
		},
	}
}
