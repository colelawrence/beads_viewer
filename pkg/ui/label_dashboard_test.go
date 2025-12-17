package ui

import (
	"testing"

	"github.com/Dicklesworthstone/beads_viewer/pkg/analysis"
	tea "github.com/charmbracelet/bubbletea"
)

func TestLabelDashboardModel_ScrollAndHomeEnd(t *testing.T) {
	m := NewLabelDashboardModel(Theme{})
	// height=3 -> visibleRows=2 (header + 2 rows)
	m.SetSize(80, 3)
	m.SetData([]analysis.LabelHealth{
		{Label: "a", HealthLevel: analysis.HealthLevelHealthy, Blocked: 0, Health: 90},
		{Label: "b", HealthLevel: analysis.HealthLevelHealthy, Blocked: 0, Health: 80},
		{Label: "c", HealthLevel: analysis.HealthLevelHealthy, Blocked: 0, Health: 70},
		{Label: "d", HealthLevel: analysis.HealthLevelHealthy, Blocked: 0, Health: 60},
		{Label: "e", HealthLevel: analysis.HealthLevelHealthy, Blocked: 0, Health: 50},
	})

	// Move cursor down within visible range; no scroll yet.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 1 || m.scrollOffset != 0 {
		t.Fatalf("after j: cursor=%d scroll=%d; want cursor=1 scroll=0", m.cursor, m.scrollOffset)
	}

	// Move down past bottom; should scroll.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 2 || m.scrollOffset != 1 {
		t.Fatalf("after j,j: cursor=%d scroll=%d; want cursor=2 scroll=1", m.cursor, m.scrollOffset)
	}

	// Move back up past top; should scroll up.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 1 || m.scrollOffset != 1 {
		t.Fatalf("after k: cursor=%d scroll=%d; want cursor=1 scroll=1", m.cursor, m.scrollOffset)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 0 || m.scrollOffset != 0 {
		t.Fatalf("after k,k: cursor=%d scroll=%d; want cursor=0 scroll=0", m.cursor, m.scrollOffset)
	}

	// End should jump to last item and scroll to bottom.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if m.cursor != 4 {
		t.Fatalf("after G: cursor=%d; want 4", m.cursor)
	}
	if m.scrollOffset != 3 {
		t.Fatalf("after G: scroll=%d; want 3", m.scrollOffset)
	}

	// Home should reset.
	m.Update(tea.KeyMsg{Type: tea.KeyHome})
	if m.cursor != 0 || m.scrollOffset != 0 {
		t.Fatalf("after home: cursor=%d scroll=%d; want cursor=0 scroll=0", m.cursor, m.scrollOffset)
	}
}

func TestLabelDashboardModel_EnterReturnsSelectedLabel(t *testing.T) {
	m := NewLabelDashboardModel(Theme{})
	m.SetSize(80, 3)
	m.SetData([]analysis.LabelHealth{
		{Label: "backend", HealthLevel: analysis.HealthLevelWarning, Blocked: 1, Health: 60},
		{Label: "frontend", HealthLevel: analysis.HealthLevelHealthy, Blocked: 0, Health: 90},
	})

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	label, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if label != "frontend" {
		t.Fatalf("enter label=%q; want %q", label, "frontend")
	}
}
