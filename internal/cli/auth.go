package cli

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/pkg/integrations/github"
	"github.com/matzehuels/stacktower/pkg/session"
)

// sessionTTL is the duration for CLI sessions (30 days).
const sessionTTL = 30 * 24 * time.Hour

// githubCommand creates the github command with subcommands.
func (c *CLI) githubCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "github",
		Short: "GitHub integration commands",
		Long: `Authenticate with GitHub and interact with your repositories.

Use the device flow to authenticate without needing a web browser callback.
Your session is stored in ~/.config/stacktower/sessions/`,
	}

	cmd.AddCommand(c.githubLoginCommand())
	cmd.AddCommand(c.githubLogoutCommand())
	cmd.AddCommand(c.githubWhoamiCommand())

	return cmd
}

// githubLoginCommand creates the login subcommand.
func (c *CLI) githubLoginCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with GitHub using device flow",
		Long: `Start the GitHub device authorization flow.

You'll be given a code to enter at https://github.com/login/device.
Once authorized, your session will be saved locally for future commands.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if existing, _ := loadGitHubSession(ctx); existing != nil {
				printInfo("Already logged in as @%s", existing.User.Login)
				printDetail("Run 'stacktower github logout' first to re-authenticate")
				return nil
			}

			_, err := c.runGitHubLogin(ctx)
			return err
		},
	}
}

// githubLogoutCommand creates the logout subcommand.
func (c *CLI) githubLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored GitHub credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := deleteGitHubSession(cmd.Context()); err != nil {
				return fmt.Errorf("delete session: %w", err)
			}
			printSuccess("Logged out")
			return nil
		},
	}
}

// githubWhoamiCommand creates the whoami subcommand.
func (c *CLI) githubWhoamiCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the currently authenticated GitHub user",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sess, err := loadGitHubSession(ctx)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			spinner := newSpinnerWithContext(ctx, "Verifying session...")
			spinner.Start()

			client := github.NewContentClient(sess.AccessToken)
			user, err := client.FetchUser(ctx)
			if err != nil {
				spinner.StopWithError("Session invalid")
				return fmt.Errorf("verify session: %w", err)
			}
			spinner.Stop()

			printSuccess("GitHub Session")
			printKeyValue("Username", "@"+user.Login)
			if user.Name != "" {
				printKeyValue("Name", user.Name)
			}
			if user.Email != "" {
				printKeyValue("Email", user.Email)
			}
			printKeyValue("Logged in", sess.CreatedAt.Format("Jan 2, 2006"))
			printKeyValue("Expires", sess.ExpiresAt.Format("Jan 2, 2006"))

			return nil
		},
	}
}

// =============================================================================
// Session Management
// =============================================================================

// loadGitHubSession loads the GitHub session from disk.
func loadGitHubSession(ctx context.Context) (*session.Session, error) {
	store, err := session.NewCLIStore()
	if err != nil {
		return nil, fmt.Errorf("open session store: %w", err)
	}

	sess, err := store.GetSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if sess == nil {
		return nil, fmt.Errorf("not logged in (run 'stacktower github login' first)")
	}

	return sess, nil
}

func saveGitHubSession(ctx context.Context, token *github.OAuthToken, user *github.User) (*session.Session, error) {
	store, err := session.NewCLIStore()
	if err != nil {
		return nil, fmt.Errorf("open session store: %w", err)
	}

	sess, err := session.New(token.AccessToken, user, sessionTTL)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	if err := store.SaveSession(ctx, sess); err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}

	return sess, nil
}

func deleteGitHubSession(ctx context.Context) error {
	store, err := session.NewCLIStore()
	if err != nil {
		return fmt.Errorf("open session store: %w", err)
	}
	return store.DeleteSession(ctx)
}

// =============================================================================
// Device Flow Login
// =============================================================================

func (c *CLI) runGitHubLogin(ctx context.Context) (*session.Session, error) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	if clientID == "" {
		clientID = github.DefaultClientID
	}

	oauthClient := github.NewOAuthClient(github.OAuthConfig{ClientID: clientID})

	loginCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	deviceResp, err := oauthClient.RequestDeviceCode(loginCtx)
	if err != nil {
		return nil, fmt.Errorf("request device code: %w", err)
	}

	printNewline()
	fmt.Println(StyleTitle.Render("GitHub Device Authorization"))
	printNewline()
	printKeyValue("Code", StyleNumber.Render(deviceResp.UserCode))
	printKeyValue("URL", StyleLink.Render(deviceResp.VerificationURI))
	printNewline()

	if err := openBrowser(deviceResp.VerificationURI); err != nil {
		printDetail("Copy the URL above and paste it in your browser")
	} else {
		printDetail("Opening browser...")
	}
	printInline("Waiting for authorization...")

	token, err := oauthClient.PollForToken(loginCtx, deviceResp.DeviceCode, deviceResp.Interval)
	if err != nil {
		fmt.Println()
		return nil, fmt.Errorf("authorization failed: %w", err)
	}

	contentClient := github.NewContentClient(token.AccessToken)
	user, err := contentClient.FetchUser(loginCtx)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}

	sess, err := saveGitHubSession(ctx, token, user)
	if err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}

	fmt.Println()
	printSuccess("Logged in as @%s", user.Login)

	return sess, nil
}

func openBrowser(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("URL scheme must be http or https, got %q", parsed.Scheme)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", rawURL)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}
