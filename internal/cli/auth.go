package cli

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/stacktower-io/stacktower/internal/cli/ui"
	"github.com/stacktower-io/stacktower/pkg/buildinfo"
	"github.com/stacktower-io/stacktower/pkg/integrations/github"
	"github.com/stacktower-io/stacktower/pkg/session"
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
	cmd.AddCommand(c.githubInstallCommand())
	cmd.AddCommand(c.githubUninstallCommand())

	return cmd
}

// githubLoginCommand creates the login subcommand.
func (c *CLI) githubLoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with GitHub using device flow",
		Long: `Start the GitHub device authorization flow.

You'll be given a code to enter at https://github.com/login/device.
Once authorized, your session will be saved locally for future commands.

Repository access is configured when you install the GitHub App.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if existing, _ := loadGitHubSession(ctx); existing != nil {
				ui.PrintInfo("Already logged in as @%s", existing.User.Login)
				ui.PrintDetail("Run 'stacktower github logout' first to re-authenticate")
				return nil
			}

			_, err := c.runGitHubLogin(ctx)
			return err
		},
	}

	return cmd
}

// githubLogoutCommand creates the logout subcommand.
func (c *CLI) githubLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored GitHub credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := deleteGitHubSession(cmd.Context()); err != nil {
				return WrapSystemError(err, "failed to delete session", "Check file permissions for ~/.config/stacktower/sessions/")
			}
			ui.PrintSuccess("Logged out")
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

			spinner := ui.NewSpinnerWithContext(ctx, "Verifying session...")
			spinner.Start()

			client := github.NewContentClient(sess.AccessToken)
			user, err := client.FetchUser(ctx)
			if err != nil {
				spinner.StopWithError("Session invalid")
				return WrapSystemError(err, "failed to verify GitHub session", "Your session may have expired. Try 'stacktower github logout' and re-login.")
			}
			spinner.Stop()

			ui.PrintHeader("GitHub Session")
			ui.PrintKeyValue("Username", "@"+user.Login)
			if user.Name != "" {
				ui.PrintKeyValue("Name", user.Name)
			}
			if user.Email != "" {
				ui.PrintKeyValue("Email", user.Email)
			}
			ui.PrintKeyValue("Logged in", sess.CreatedAt.Format("Jan 2, 2006"))
			ui.PrintKeyValue("Expires", sess.ExpiresAt.Format("Jan 2, 2006"))

			// Check app installation status
			installation, err := client.HasAppInstallation(ctx, buildinfo.GitHubAppSlug)
			if err == nil {
				ui.PrintNewline()
				if installation != nil {
					ui.PrintKeyValue("App Status", ui.StyleSuccess.Render("Installed")+" (@"+installation.Account.Login+")")
				} else {
					ui.PrintKeyValue("App Status", ui.StyleWarning.Render("Not installed"))
					ui.PrintDetail("Run 'stacktower github install' to install the app")
				}
			}

			return nil
		},
	}
}

// githubInstallCommand creates the install subcommand.
func (c *CLI) githubInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install or manage the Stacktower GitHub App",
		Long: `Open the GitHub App installation page in your browser.

This allows you to install the Stacktower app on your account or organization,
and configure which repositories it can access.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Check if logged in
			sess, err := loadGitHubSession(ctx)
			if err != nil {
				ui.PrintWarning("Not logged in. Run 'stacktower github login' first.")
				return nil
			}

			// Check current installation status
			client := github.NewContentClient(sess.AccessToken)
			installation, err := client.HasAppInstallation(ctx, buildinfo.GitHubAppSlug)
			if err != nil {
				c.Logger.Debug("failed to check app installation", "error", err)
			}

			installURL := fmt.Sprintf("https://github.com/apps/%s/installations/new", buildinfo.GitHubAppSlug)

			if installation != nil {
				ui.PrintInfo("App already installed for @%s", installation.Account.Login)
				ui.PrintDetail("Opening settings to manage installation...")
				// Link to settings to manage the installation
				installURL = "https://github.com/settings/installations"
			} else {
				ui.PrintInfo("Opening GitHub App installation page...")
			}

			ui.PrintKeyValue("URL", ui.StyleLink.Render(installURL))

			if err := openBrowser(installURL); err != nil {
				ui.PrintDetail("Copy the URL above and paste it in your browser")
			}

			return nil
		},
	}
}

// githubUninstallCommand creates the uninstall subcommand.
func (c *CLI) githubUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the Stacktower GitHub App",
		Long: `Open the GitHub App settings page to uninstall Stacktower.

This removes Stacktower's access to your repositories. You can re-install
at any time with 'stacktower github install'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			sess, err := loadGitHubSession(ctx)
			if err != nil {
				ui.PrintWarning("Not logged in. Run 'stacktower github login' first.")
				return nil
			}

			client := github.NewContentClient(sess.AccessToken)
			installation, err := client.HasAppInstallation(ctx, buildinfo.GitHubAppSlug)
			if err != nil {
				c.Logger.Debug("failed to check app installation", "error", err)
			}

			if installation == nil {
				ui.PrintInfo("GitHub App is not installed")
				ui.PrintDetail("Run 'stacktower github install' to install it")
				return nil
			}

			uninstallURL := fmt.Sprintf("https://github.com/settings/installations/%d", installation.ID)

			ui.PrintNewline()
			ui.PrintHeader("Uninstall Stacktower GitHub App")
			ui.PrintKeyValue("Installed on", "@"+installation.Account.Login)
			ui.PrintKeyValue("URL", ui.StyleLink.Render(uninstallURL))
			ui.PrintNewline()
			ui.PrintDetail("The settings page will open in your browser.")
			ui.PrintDetail("Scroll to \"Danger zone\" and click Uninstall.")
			ui.PrintNewline()

			if err := openBrowser(uninstallURL); err != nil {
				ui.PrintDetail("Copy the URL above and paste it in your browser")
			}

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
		return nil, WrapSystemError(err, "failed to open session store", "Check file permissions for ~/.config/stacktower/sessions/")
	}

	sess, err := store.GetSession(ctx)
	if err != nil {
		return nil, WrapSystemError(err, "failed to read session", "Try 'stacktower github logout' and re-login.")
	}
	if sess == nil {
		return nil, NewUserError("not logged in", "Run 'stacktower github login' first.")
	}

	return sess, nil
}

func saveGitHubSession(ctx context.Context, token *github.OAuthToken, user *github.User) (*session.Session, error) {
	store, err := session.NewCLIStore()
	if err != nil {
		return nil, WrapSystemError(err, "failed to open session store", "Check file permissions for ~/.config/stacktower/sessions/")
	}

	sess, err := session.New(token.AccessToken, user, sessionTTL)
	if err != nil {
		return nil, WrapSystemError(err, "failed to create session", "")
	}

	if err := store.SaveSession(ctx, sess); err != nil {
		return nil, WrapSystemError(err, "failed to save session", "Check file permissions for ~/.config/stacktower/sessions/")
	}

	return sess, nil
}

func deleteGitHubSession(ctx context.Context) error {
	store, err := session.NewCLIStore()
	if err != nil {
		return WrapSystemError(err, "failed to open session store", "Check file permissions for ~/.config/stacktower/sessions/")
	}
	if err := store.DeleteSession(ctx); err != nil {
		return WrapSystemError(err, "failed to delete session", "Check file permissions for ~/.config/stacktower/sessions/")
	}
	return nil
}

// =============================================================================
// Device Flow Login
// =============================================================================

func (c *CLI) runGitHubLogin(ctx context.Context) (*session.Session, error) {
	if buildinfo.GitHubAppClientID == "" {
		return nil, NewSystemError("GitHub login not available in this build", "This binary was built without a GitHub App client ID.")
	}

	oauthClient := github.NewOAuthClient(github.OAuthConfig{ClientID: buildinfo.GitHubAppClientID})

	loginCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	deviceResp, err := oauthClient.RequestDeviceCode(loginCtx)
	if err != nil {
		return nil, WrapSystemError(err, "failed to request device code", "Check your network connection and try again.")
	}

	ui.PrintNewline()
	ui.PrintHeader("GitHub Device Authorization")
	ui.PrintKeyValue("Code", ui.StyleNumber.Render(deviceResp.UserCode))
	ui.PrintKeyValue("URL", ui.StyleLink.Render(deviceResp.VerificationURI))
	ui.PrintNewline()

	if err := openBrowser(deviceResp.VerificationURI); err != nil {
		ui.PrintDetail("Copy the URL above and paste it in your browser")
	} else {
		ui.PrintDetail("Opening browser...")
	}

	spinner := ui.NewSpinnerWithContext(loginCtx, "Waiting for authorization...")
	spinner.Start()

	token, err := oauthClient.PollForToken(loginCtx, deviceResp.DeviceCode, deviceResp.Interval)
	if err != nil {
		spinner.StopWithError("Authorization failed")
		return nil, WrapSystemError(err, "authorization failed", "Make sure you entered the code at the URL above and approved the request.")
	}
	spinner.Stop()

	contentClient := github.NewContentClient(token.AccessToken)
	user, err := contentClient.FetchUser(loginCtx)
	if err != nil {
		return nil, WrapSystemError(err, "failed to fetch GitHub user info", "Authorization succeeded but user lookup failed. Try again.")
	}

	sess, err := saveGitHubSession(ctx, token, user)
	if err != nil {
		return nil, err // already wrapped by saveGitHubSession
	}

	ui.PrintNewline()
	ui.PrintSuccess("Logged in as @%s", user.Login)

	// Check for GitHub App installation
	installation, err := contentClient.HasAppInstallation(loginCtx, buildinfo.GitHubAppSlug)
	if err != nil {
		c.Logger.Debug("failed to check app installation", "error", err)
	} else if installation == nil {
		// App not installed - prompt user to install
		ui.PrintNewline()
		ui.PrintWarning("GitHub App not installed")
		ui.PrintDetail("To access your repositories, install the Stacktower app:")

		installURL := fmt.Sprintf("https://github.com/apps/%s/installations/new", buildinfo.GitHubAppSlug)
		ui.PrintKeyValue("URL", ui.StyleLink.Render(installURL))
		ui.PrintNewline()

		if err := openBrowser(installURL); err != nil {
			ui.PrintDetail("Copy the URL above and paste it in your browser")
		} else {
			ui.PrintDetail("Opening browser to install the app...")
		}
	} else {
		ui.PrintDetail("GitHub App installed for @%s", installation.Account.Login)
	}

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
