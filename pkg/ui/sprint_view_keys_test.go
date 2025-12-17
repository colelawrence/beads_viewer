package ui

import (
	"testing"
	"time"

	"github.com/Dicklesworthstone/beads_viewer/pkg/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestHandleSprintKeys_Exit(t *testing.T) {
	m := Model{
		isSprintView: true,
		focused:      focusDetail,
		theme:        DefaultTheme(lipgloss.NewRenderer(nil)),
		width:        100,
		height:       40,
	}
	m = m.handleSprintKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P")})
	if m.isSprintView {
		t.Fatalf("expected sprint view to exit")
	}
	if m.focused != focusList {
		t.Fatalf("focused=%v; want focusList", m.focused)
	}
}

func TestHandleSprintKeys_NextPrevSprint(t *testing.T) {
	now := time.Now().UTC()
	sprints := []model.Sprint{
		{ID: "s1", Name: "Sprint 1", StartDate: now.AddDate(0, 0, -7), EndDate: now.AddDate(0, 0, -1), BeadIDs: []string{"A"}},
		{ID: "s2", Name: "Sprint 2", StartDate: now.AddDate(0, 0, -1), EndDate: now.AddDate(0, 0, 7), BeadIDs: []string{"A"}},
	}

	m := Model{
		isSprintView:   true,
		theme:          DefaultTheme(lipgloss.NewRenderer(nil)),
		width:          100,
		height:         40,
		issues:         []model.Issue{{ID: "A", Title: "Issue A", Status: model.StatusOpen, Priority: 1, IssueType: model.TypeTask}},
		sprints:        sprints,
		selectedSprint: &sprints[0],
	}

	m = m.handleSprintKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.selectedSprint == nil || m.selectedSprint.ID != "s2" {
		t.Fatalf("after j: selected=%v; want s2", m.selectedSprint)
	}
	if m.sprintViewText == "" {
		t.Fatalf("expected sprintViewText to be populated after navigation")
	}

	m = m.handleSprintKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.selectedSprint == nil || m.selectedSprint.ID != "s1" {
		t.Fatalf("after k: selected=%v; want s1", m.selectedSprint)
	}
}
