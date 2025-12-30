package cli

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	"github.com/matzehuels/stacktower/internal/cli/term"
	"github.com/matzehuels/stacktower/pkg/infra"
	"github.com/matzehuels/stacktower/pkg/infra/session"
	"github.com/matzehuels/stacktower/pkg/integrations/github"
)

// cliSessionTTL is the duration for CLI sessions.
// 30 days balances user convenience (avoiding frequent re-authentication via device flow)
// with security (limiting exposure window if credentials are compromised).
// Users can re-authenticate with 'stacktower github login' when sessions expire.
const cliSessionTTL = 30 * 24 * time.Hour

// getCLIStore returns the CLI session store.
func getCLIStore() (*session.CLIStore, error) {
	return session.NewCLIStore()
}

// LoadGitHubSession loads the GitHub session from disk.
// Returns nil with an error message if not logged in.
func LoadGitHubSession(ctx context.Context) (*session.Session, error) {
	store, err := getCLIStore()
	if err != nil {
		return nil, fmt.Errorf("session store: %w", err)
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

// saveGitHubSession saves the GitHub session to disk and returns it.
// Returns the session so callers can use it directly without re-reading from disk.
func saveGitHubSession(ctx context.Context, token *github.OAuthToken, user *github.User) (*session.Session, error) {
	store, err := getCLIStore()
	if err != nil {
		return nil, fmt.Errorf("session store: %w", err)
	}

	sess, err := session.New(token.AccessToken, user, cliSessionTTL)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	if err := store.SaveSession(ctx, sess); err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}

	return sess, nil
}

// deleteGitHubSession removes the stored session.
func deleteGitHubSession(ctx context.Context) error {
	store, err := getCLIStore()
	if err != nil {
		return fmt.Errorf("session store: %w", err)
	}
	return store.DeleteSession(ctx)
}

// openBrowser opens the specified URL in the default browser.
// It validates the URL scheme to prevent command injection via malicious URLs.
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

// runGitHubLogin performs the GitHub device flow login and returns the session.
// This can be called from any command that needs authentication.
func runGitHubLogin(ctx context.Context) (*session.Session, error) {
	cfg := infra.LoadGitHubConfig()
	clientID := cfg.ClientID
	if clientID == "" {
		clientID = github.DefaultClientID
	}

	oauthClient := github.NewOAuthClient(github.OAuthConfig{
		ClientID: clientID,
	})

	// Request device code
	loginCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	deviceResp, err := oauthClient.RequestDeviceCode(loginCtx)
	if err != nil {
		return nil, fmt.Errorf("request device code: %w", err)
	}

	// Show authorization info
	term.PrintNewline()
	fmt.Println(term.StyleTitle.Render("GitHub Device Authorization"))
	term.PrintNewline()
	term.PrintKeyValue("Code", term.StyleNumber.Render(deviceResp.UserCode))
	term.PrintKeyValue("URL", term.StyleLink.Render(deviceResp.VerificationURI))
	term.PrintNewline()

	// Try to open browser automatically
	if err := openBrowser(deviceResp.VerificationURI); err != nil {
		term.PrintDetail("Copy the URL above and paste it in your browser")
	} else {
		term.PrintDetail("Opening browser...")
	}
	term.PrintInline("Waiting for authorization...")

	// Poll for token
	token, err := oauthClient.PollForToken(loginCtx, deviceResp.DeviceCode, deviceResp.Interval)
	if err != nil {
		fmt.Println()
		return nil, fmt.Errorf("authorization failed: %w", err)
	}

	// Fetch user info
	contentClient := github.NewContentClient(token.AccessToken)
	user, err := contentClient.FetchUser(loginCtx)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}

	// Save session and return it directly (avoids unnecessary disk read)
	sess, err := saveGitHubSession(ctx, token, user)
	if err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}

	fmt.Println()
	term.PrintSuccess("Logged in as @%s", user.Login)

	return sess, nil
}
