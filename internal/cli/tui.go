package cli

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/matzehuels/stacktower/pkg/core/deps"
	"github.com/matzehuels/stacktower/pkg/core/deps/languages"
	"github.com/matzehuels/stacktower/pkg/integrations/github"
)

// List styles
var (
	listSelectedStyle = lipgloss.NewStyle().Bold(true).Foreground(colorCyan)
	listNormalStyle   = lipgloss.NewStyle().Foreground(colorWhite)
	listDimStyle      = lipgloss.NewStyle().Foreground(colorDim)
)

// =============================================================================
// RepoListModel - Interactive repository selection
// =============================================================================

// RepoSelection holds the result of the repo selection.
type RepoSelection struct {
	Repo *github.RepoWithManifests
}

// RepoListModel is the bubbletea model for interactive repo selection.
type RepoListModel struct {
	Repos    []github.RepoWithManifests
	Cursor   int
	Selected *RepoSelection
	Height   int
	Offset   int
}

// NewRepoListModel creates a new repo list model.
func NewRepoListModel(repos []github.RepoWithManifests) RepoListModel {
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
				if m.Cursor < m.Offset {
					m.Offset = m.Cursor
				}
			}
		case "down", "j":
			if m.Cursor < len(m.Repos)-1 {
				m.Cursor++
				if m.Cursor >= m.Offset+m.Height {
					m.Offset = m.Cursor - m.Height + 1
				}
			}
		case "enter":
			repo := m.Repos[m.Cursor]
			if len(repo.Manifests) == 0 {
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

	b.WriteString(StyleTitle.Render("Select Repository"))
	b.WriteString("\n")
	b.WriteString(listDimStyle.Render("↑/↓ navigate  ⏎ select  q quit"))
	b.WriteString("\n\n")

	end := m.Offset + m.Height
	if end > len(m.Repos) {
		end = len(m.Repos)
	}

	rows := [][]string{}
	for i := m.Offset; i < end; i++ {
		r := m.Repos[i]
		hasManifests := len(r.Manifests) > 0

		cursor := "  "
		if i == m.Cursor {
			cursor = "▸ "
		}

		visibility := "✓"
		if r.Repo.Private {
			visibility = ""
		}

		lang := ""
		if hasManifests {
			lang = r.Manifests[0].Language
		} else if r.Repo.Language != "" {
			lang = deps.NormalizeLanguageName(r.Repo.Language, languages.All)
		}
		if lang == "" {
			lang = "—"
		}

		manifestStr := "—"
		if hasManifests {
			names := make([]string, len(r.Manifests))
			for j, mf := range r.Manifests {
				names[j] = mf.Name
			}
			manifestStr = strings.Join(names, ", ")
		}

		updated := formatRelativeTime(r.Repo.UpdatedAt)
		rows = append(rows, []string{cursor, r.Repo.FullName, lang, visibility, updated, manifestStr})
	}

	headerStyle := lipgloss.NewStyle().Foreground(colorGray).Bold(true)

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorDim)).
		Headers("", "Repository", "Lang", "Public", "Updated", "Manifests").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
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

			base := lipgloss.NewStyle()
			if col == 3 || col == 4 {
				if isCurrent {
					base = base.Foreground(colorGray)
				} else {
					base = base.Foreground(colorDim)
				}
			}

			if isCurrent {
				if hasManifests {
					if col != 3 && col != 4 {
						return base.Foreground(colorGreen).Bold(true)
					}
					return base.Bold(true)
				}
				return base.Foreground(colorDim).Bold(true)
			} else if hasManifests {
				if col != 3 && col != 4 {
					return base.Foreground(colorGreen)
				}
				return base
			}
			return base.Foreground(colorDim)
		})

	b.WriteString(t.Render())
	b.WriteString("\n\n")
	b.WriteString(listDimStyle.Render(fmt.Sprintf("  [%d/%d]", m.Cursor+1, len(m.Repos))))

	return b.String()
}

// =============================================================================
// ManifestListModel - Interactive manifest file selection
// =============================================================================

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

	b.WriteString(StyleTitle.Render("Select Manifest File"))
	b.WriteString("\n")
	b.WriteString(listDimStyle.Render("arrows: navigate  enter: select  q: quit"))
	b.WriteString("\n\n")

	for i, mf := range m.Manifests {
		cursor := "  "
		if i == m.Cursor {
			cursor = "> "
		}

		supported := deps.IsManifestSupported(mf.Name, languages.All)
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

	b.WriteString("\n")
	b.WriteString(listDimStyle.Render(strings.Repeat("-", 40)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s supported   %s not yet supported\n",
		StyleSuccess.Render("*"), StyleWarning.Render("!")))

	return b.String()
}

// =============================================================================
// Helpers
// =============================================================================

func formatRelativeTime(s string) string {
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
