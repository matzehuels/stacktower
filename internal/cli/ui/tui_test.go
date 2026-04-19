package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stacktower-io/stacktower/pkg/integrations/github"
)

func TestRefListModel_FilterAsYouType(t *testing.T) {
	m := NewRefListModel(
		[]github.Branch{
			{Name: "main", Commit: "aaa"},
			{Name: "feature/auth", Commit: "bbb"},
			{Name: "feature/api", Commit: "ccc"},
			{Name: "fix/typo", Commit: "ddd"},
		},
		[]github.Tag{
			{Name: "v1.0.0", Commit: "eee"},
			{Name: "v2.0.0", Commit: "fff"},
		},
		"main",
	)

	// All items should be visible initially
	if len(m.Filtered) != 6 {
		t.Fatalf("initial Filtered = %d, want 6", len(m.Filtered))
	}
	if m.Filter != "" {
		t.Fatalf("initial Filter = %q, want empty", m.Filter)
	}

	// Type "feat" — should filter to feature/auth and feature/api
	for _, ch := range "feat" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	if m.Filter != "feat" {
		t.Fatalf("after typing 'feat': Filter = %q, want %q", m.Filter, "feat")
	}
	if len(m.Filtered) != 2 {
		t.Fatalf("after typing 'feat': Filtered = %d, want 2", len(m.Filtered))
	}

	// Verify the filtered items are the correct ones
	for _, idx := range m.Filtered {
		name := m.Items[idx].Name
		if name != "feature/auth" && name != "feature/api" {
			t.Errorf("unexpected filtered item: %q", name)
		}
	}

	// Backspace to "fea" — should still match same two
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.Filter != "fea" {
		t.Fatalf("after backspace: Filter = %q, want %q", m.Filter, "fea")
	}
	if len(m.Filtered) != 2 {
		t.Fatalf("after backspace: Filtered = %d, want 2", len(m.Filtered))
	}

	// Type "ture/au" to narrow to just feature/auth
	for _, ch := range "ture/au" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	if m.Filter != "feature/au" {
		t.Fatalf("after full filter: Filter = %q, want %q", m.Filter, "feature/au")
	}
	if len(m.Filtered) != 1 {
		t.Fatalf("after full filter: Filtered = %d, want 1", len(m.Filtered))
	}
	if m.Items[m.Filtered[0]].Name != "feature/auth" {
		t.Errorf("filtered item = %q, want feature/auth", m.Items[m.Filtered[0]].Name)
	}
}

func TestRefListModel_FilterNoMatches(t *testing.T) {
	m := NewRefListModel(
		[]github.Branch{{Name: "main", Commit: "aaa"}},
		nil,
		"main",
	)

	for _, ch := range "zzz" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}

	if len(m.Filtered) != 0 {
		t.Fatalf("Filtered = %d, want 0", len(m.Filtered))
	}
	if m.Cursor != 0 {
		t.Fatalf("Cursor = %d, want 0 (clamped)", m.Cursor)
	}
}

func TestRefListModel_CursorClampsOnFilter(t *testing.T) {
	m := NewRefListModel(
		[]github.Branch{
			{Name: "main", Commit: "aaa"},
			{Name: "develop", Commit: "bbb"},
			{Name: "feature/x", Commit: "ccc"},
		},
		nil,
		"main",
	)

	// Move cursor to last item
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.Cursor != 2 {
		t.Fatalf("Cursor = %d, want 2", m.Cursor)
	}

	// Filter to just "main" — cursor should clamp to 0
	for _, ch := range "main" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	if len(m.Filtered) != 1 {
		t.Fatalf("Filtered = %d, want 1", len(m.Filtered))
	}
	if m.Cursor != 0 {
		t.Errorf("Cursor = %d, want 0 (clamped)", m.Cursor)
	}
}

func TestRefListModel_SelectEnter(t *testing.T) {
	m := NewRefListModel(
		[]github.Branch{
			{Name: "main", Commit: "aaa"},
			{Name: "develop", Commit: "bbb"},
		},
		nil,
		"main",
	)

	// Move to "develop" and press enter
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.Selected == nil {
		t.Fatal("Selected should not be nil after enter")
	}
	if m.Selected.Name != "develop" {
		t.Errorf("Selected.Name = %q, want develop", m.Selected.Name)
	}
}

func TestRefListModel_DefaultBranchFirst(t *testing.T) {
	m := NewRefListModel(
		[]github.Branch{
			{Name: "develop", Commit: "bbb"},
			{Name: "main", Commit: "aaa"},
			{Name: "feature", Commit: "ccc"},
		},
		nil,
		"main",
	)

	if len(m.Items) != 3 {
		t.Fatalf("Items = %d, want 3", len(m.Items))
	}
	if m.Items[0].Name != "main" {
		t.Errorf("first item = %q, want main (default branch)", m.Items[0].Name)
	}
	if !m.Items[0].IsDefault {
		t.Error("first item should be marked as default")
	}
}

func TestRefListModel_FilterIsCaseInsensitive(t *testing.T) {
	m := NewRefListModel(
		[]github.Branch{
			{Name: "Main", Commit: "aaa"},
			{Name: "DEVELOP", Commit: "bbb"},
		},
		nil,
		"Main",
	)

	for _, ch := range "dev" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}

	if len(m.Filtered) != 1 {
		t.Fatalf("Filtered = %d, want 1", len(m.Filtered))
	}
	if m.Items[m.Filtered[0]].Name != "DEVELOP" {
		t.Errorf("filtered item = %q, want DEVELOP", m.Items[m.Filtered[0]].Name)
	}
}
