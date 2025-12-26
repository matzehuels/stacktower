package cli

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/pkg/deps"
	"github.com/matzehuels/stacktower/pkg/deps/languages"
	"github.com/matzehuels/stacktower/pkg/infra"
	"github.com/matzehuels/stacktower/pkg/integrations/github"
	"github.com/matzehuels/stacktower/pkg/session"
)

// cliSessionTTL is the duration for CLI sessions (long-lived for convenience).
const cliSessionTTL = 90 * 24 * time.Hour // 90 days

// getCLIStore returns the CLI session store.
func getCLIStore() (*session.CLIStore, error) {
	return session.NewCLIStore()
}

// LoadGitHubSession loads the GitHub session from disk.
// Returns nil with an error message if not logged in.
func LoadGitHubSession() (*session.Session, error) {
	store, err := getCLIStore()
	if err != nil {
		return nil, fmt.Errorf("session store: %w", err)
	}

	sess, err := store.GetSession(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if sess == nil {
		return nil, fmt.Errorf("not logged in (run 'stacktower github login' first)")
	}

	return sess, nil
}

// saveGitHubSession saves the GitHub session to disk.
func saveGitHubSession(token *github.OAuthToken, user *github.User) error {
	store, err := getCLIStore()
	if err != nil {
		return fmt.Errorf("session store: %w", err)
	}

	sess, err := session.New(token.AccessToken, user, cliSessionTTL)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	if err := store.SaveSession(context.Background(), sess); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	return nil
}

// deleteGitHubSession removes the stored session.
func deleteGitHubSession() error {
	store, err := getCLIStore()
	if err != nil {
		return fmt.Errorf("session store: %w", err)
	}
	return store.DeleteSession(context.Background())
}

// openBrowser opens the specified URL in the default browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
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

	// Build the content using shared styles
	content := fmt.Sprintf(
		"%s\n\n%s  %s\n\n%s  %s",
		StyleTitle.Render("GitHub Device Authorization"),
		StyleLabel.Render("Code:"),
		StyleHighlight.Render(deviceResp.UserCode),
		StyleLabel.Render("URL:"),
		StyleLink.Render(deviceResp.VerificationURI),
	)

	fmt.Println(StyleBox.Render(content))

	// Try to open browser automatically
	if err := openBrowser(deviceResp.VerificationURI); err != nil {
		fmt.Println(StyleDim.Render("Copy the URL above and paste it in your browser"))
	} else {
		fmt.Println(StyleDim.Render("Opening browser..."))
	}
	fmt.Println()
	fmt.Print(StyleDim.Render("Waiting for authorization..."))

	// Poll for token
	token, err := oauthClient.PollForToken(loginCtx, deviceResp.DeviceCode, deviceResp.Interval)
	if err != nil {
		return nil, fmt.Errorf("authorization failed: %w", err)
	}

	// Fetch user info
	contentClient := github.NewContentClient(token.AccessToken)
	user, err := contentClient.FetchUser(loginCtx)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}

	// Save session
	if err := saveGitHubSession(token, user); err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}

	fmt.Printf("\n%s %s\n\n", StyleSuccess.Render("Logged in as"), StyleHighlight.Render("@"+user.Login))

	// Return the session
	return LoadGitHubSession()
}

// newGitHubCmd creates the github command with subcommands.
func newGitHubCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "github",
		Short: "GitHub integration commands",
		Long: `Authenticate with GitHub and interact with your repositories.

Use the device flow to authenticate without needing a web browser callback.
Your session is stored in ~/.config/stacktower/sessions/`,
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
			// Check if already logged in
			if existing, _ := LoadGitHubSession(); existing != nil {
				fmt.Printf("Already logged in as @%s\n", existing.User.Login)
				fmt.Println("Run 'stacktower github logout' first to re-authenticate")
				return nil
			}

			_, err := runGitHubLogin(cmd.Context())
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
			if err := deleteGitHubSession(); err != nil {
				return err
			}
			fmt.Println("Logged out (session removed)")
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
			sess, err := LoadGitHubSession()
			if err != nil {
				return err
			}

			// Verify token still works by fetching fresh user info
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			client := github.NewContentClient(sess.AccessToken)
			user, err := client.FetchUser(ctx)
			if err != nil {
				return fmt.Errorf("session may be invalid: %w", err)
			}

			fmt.Println()
			fmt.Printf("  Username:  @%s\n", user.Login)
			if user.Name != "" {
				fmt.Printf("  Name:      %s\n", user.Name)
			}
			if user.Email != "" {
				fmt.Printf("  Email:     %s\n", user.Email)
			}
			fmt.Printf("  Logged in: %s\n", sess.CreatedAt.Format("Jan 2, 2006 at 3:04pm"))
			fmt.Printf("  Expires:   %s\n", sess.ExpiresAt.Format("Jan 2, 2006"))
			fmt.Println()

			return nil
		},
	}
}

// --- Bubbletea model for interactive repo list ---

// List styles - using shared color palette
var (
	listSelectedStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	listNormalStyle   = lipgloss.NewStyle().Foreground(ColorMuted)
	listDimStyle      = lipgloss.NewStyle().Foreground(ColorDim)
	listTitleStyle    = StyleTitle
)

// RepoSelection holds the result of the repo selection.
type RepoSelection struct {
	Repo *github.RepoWithManifests
}

// RepoListModel is the bubbletea model for interactive repo selection.
// Simple flat list with proper scrolling - manifests shown inline.
type RepoListModel struct {
	Repos    []github.RepoWithManifests
	Cursor   int
	Selected *RepoSelection
	Height   int
	Offset   int // For scrolling
}

// NewRepoListModel creates a new repo list model.
func NewRepoListModel(repos []github.RepoWithManifests, _ bool) RepoListModel {
	return RepoListModel{
		Repos:  repos,
		Cursor: 0,
		Height: 15,
		Offset: 0,
	}
}

func (m RepoListModel) Init() tea.Cmd {
	return nil
}

func (m RepoListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
				// Scroll up if cursor goes above visible area
				if m.Cursor < m.Offset {
					m.Offset = m.Cursor
				}
			}

		case "down", "j":
			if m.Cursor < len(m.Repos)-1 {
				m.Cursor++
				// Scroll down if cursor goes below visible area
				if m.Cursor >= m.Offset+m.Height {
					m.Offset = m.Cursor - m.Height + 1
				}
			}

		case "enter":
			repo := m.Repos[m.Cursor]
			if len(repo.Manifests) == 0 {
				// No manifests - can't select
				return m, nil
			}
			m.Selected = &RepoSelection{Repo: &repo}
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.Height = msg.Height - 6
		if m.Height < 5 {
			m.Height = 5
		}
	}
	return m, nil
}

func (m RepoListModel) View() string {
	var b strings.Builder

	b.WriteString(listTitleStyle.Render("Select Repository"))
	b.WriteString("\n")
	b.WriteString(listDimStyle.Render("↑/↓ navigate  ⏎ select  q quit"))
	b.WriteString("\n\n")

	// Calculate visible range
	end := m.Offset + m.Height
	if end > len(m.Repos) {
		end = len(m.Repos)
	}

	// Build table rows
	rows := [][]string{}
	for i := m.Offset; i < end; i++ {
		r := m.Repos[i]
		hasManifests := len(r.Manifests) > 0

		// Cursor
		cursor := "  "
		if i == m.Cursor {
			cursor = "▸ "
		}

		// Visibility (boolean: public = yes)
		visibility := "✓"
		if r.Repo.Private {
			visibility = ""
		}

		// Language (from manifest or repo)
		lang := ""
		if hasManifests {
			lang = r.Manifests[0].Language
		} else if r.Repo.Language != "" {
			lang = deps.NormalizeLanguageName(r.Repo.Language, languages.All)
		}
		if lang == "" {
			lang = "—"
		}

		// Manifest names (comma-separated)
		manifestStr := "—"
		if hasManifests {
			names := make([]string, len(r.Manifests))
			for j, mf := range r.Manifests {
				names[j] = mf.Name
			}
			manifestStr = strings.Join(names, ", ")
		}

		// Updated at (relative time)
		updated := parseGitHubTime(r.Repo.UpdatedAt)

		rows = append(rows, []string{cursor, r.Repo.FullName, lang, visibility, updated, manifestStr})
	}

	// Column styles
	headerStyle := lipgloss.NewStyle().Foreground(ColorMuted).Bold(true)

	// Create table with rounded borders
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(ColorDim)).
		Headers("", "Repository", "Lang", "Public", "Updated", "Manifests").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			// Header row
			if row == -1 {
				return headerStyle
			}

			actualIdx := m.Offset + row
			if actualIdx >= len(m.Repos) {
				return lipgloss.NewStyle()
			}
			r := m.Repos[actualIdx]
			hasManifests := len(r.Manifests) > 0
			isCurrent := actualIdx == m.Cursor

			// Base style per column type
			base := lipgloss.NewStyle()

			// Dim metadata columns (visibility, updated)
			if col == 3 || col == 4 {
				if isCurrent {
					base = base.Foreground(ColorMuted)
				} else {
					base = base.Foreground(ColorDim)
				}
			}

			// Row-level styling
			if isCurrent {
				if hasManifests {
					if col != 3 && col != 4 {
						return base.Foreground(ColorSuccess).Bold(true)
					}
					return base.Bold(true)
				}
				return base.Foreground(ColorDim).Bold(true)
			} else if hasManifests {
				if col != 3 && col != 4 {
					return base.Foreground(ColorSuccess)
				}
				return base
			}
			return base.Foreground(ColorDim)
		})

	b.WriteString(t.Render())

	// Footer with position
	b.WriteString("\n\n")
	b.WriteString(listDimStyle.Render(fmt.Sprintf("  [%d/%d]", m.Cursor+1, len(m.Repos))))

	return b.String()
}

// parseGitHubTime formats a GitHub timestamp for display.
func parseGitHubTime(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}

	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	default:
		return t.Format("Jan 2, 2006")
	}
}

// --- Manifest selection model ---

// ManifestListModel is the bubbletea model for interactive manifest selection.
type ManifestListModel struct {
	Manifests []github.ManifestFile
	Cursor    int
	Selected  *github.ManifestFile
}

// NewManifestListModel creates a new manifest list model.
func NewManifestListModel(manifests []github.ManifestFile) ManifestListModel {
	return ManifestListModel{Manifests: manifests}
}

func (m ManifestListModel) Init() tea.Cmd {
	return nil
}

func (m ManifestListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			if m.Cursor < len(m.Manifests)-1 {
				m.Cursor++
			}
		case "enter":
			m.Selected = &m.Manifests[m.Cursor]
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m ManifestListModel) View() string {
	var b strings.Builder

	b.WriteString(listTitleStyle.Render("Select Manifest File"))
	b.WriteString("\n")
	b.WriteString(listDimStyle.Render("arrows: navigate  enter: select  q: quit"))
	b.WriteString("\n\n")

	for i, mf := range m.Manifests {
		cursor := "  "
		if i == m.Cursor {
			cursor = "> "
		}

		supported := deps.IsManifestSupported(mf.Name, languages.All)

		// Status indicator
		var status string
		if supported {
			status = StyleSuccess.Render("*")
		} else {
			status = StyleWarning.Render("!")
		}

		line := fmt.Sprintf("%s%s %-25s  %s", cursor, status, mf.Name, listDimStyle.Render(mf.Language))

		if i == m.Cursor {
			b.WriteString(listSelectedStyle.Render(line))
		} else if !supported {
			b.WriteString(listDimStyle.Render(line))
		} else {
			b.WriteString(listNormalStyle.Render(line))
		}
		b.WriteString("\n")
	}

	// Legend
	b.WriteString("\n")
	b.WriteString(listDimStyle.Render(strings.Repeat("-", 40)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s supported   %s not yet supported\n",
		StyleSuccess.Render("*"), StyleWarning.Render("!")))

	return b.String()
}
