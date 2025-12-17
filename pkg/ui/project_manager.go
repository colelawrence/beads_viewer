package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// ProjectEntry represents a project in the project manager.
type ProjectEntry struct {
	Name       string // Display name
	Path       string // Absolute path to project directory
	Prefix     string // Namespace prefix (e.g., "api-")
	IssueCount int    // Number of issues from this project
	IsActive   bool   // Whether currently included in view
}

// ProjectManagerModel represents the project manager overlay.
type ProjectManagerModel struct {
	projects      []ProjectEntry
	selectedIndex int
	addMode       bool // True when entering a new path
	pathInput     textinput.Model
	width         int
	height        int
	theme         Theme
	errorMsg      string
}

// NewProjectManagerModel creates a new project manager.
func NewProjectManagerModel(theme Theme) ProjectManagerModel {
	ti := textinput.New()
	ti.Placeholder = "/path/to/project"
	ti.CharLimit = 256
	ti.Width = 50
	ti.Prompt = "Path: "

	return ProjectManagerModel{
		projects:  []ProjectEntry{},
		pathInput: ti,
		theme:     theme,
	}
}

// SetProjects sets the project list.
func (m *ProjectManagerModel) SetProjects(projects []ProjectEntry) {
	m.projects = projects
	if m.selectedIndex >= len(projects) {
		m.selectedIndex = len(projects) - 1
	}
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}
}

// SetSize updates the overlay dimensions.
func (m *ProjectManagerModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Resize text input
	inputWidth := 50
	if width < 70 {
		inputWidth = width - 20
	}
	if inputWidth < 30 {
		inputWidth = 30
	}
	m.pathInput.Width = inputWidth
}

// MoveUp moves selection up.
func (m *ProjectManagerModel) MoveUp() {
	if m.addMode {
		return
	}
	if m.selectedIndex > 0 {
		m.selectedIndex--
	}
}

// MoveDown moves selection down.
func (m *ProjectManagerModel) MoveDown() {
	if m.addMode {
		return
	}
	if m.selectedIndex < len(m.projects)-1 {
		m.selectedIndex++
	}
}

// ToggleActive toggles whether the selected project is active.
func (m *ProjectManagerModel) ToggleActive() {
	if m.addMode || len(m.projects) == 0 {
		return
	}
	if m.selectedIndex >= 0 && m.selectedIndex < len(m.projects) {
		m.projects[m.selectedIndex].IsActive = !m.projects[m.selectedIndex].IsActive
	}
}

// EnterAddMode enters the mode for adding a new project path.
func (m *ProjectManagerModel) EnterAddMode() {
	m.addMode = true
	m.pathInput.SetValue("")
	m.pathInput.Focus()
	m.errorMsg = ""
}

// ExitAddMode exits add mode without adding.
func (m *ProjectManagerModel) ExitAddMode() {
	m.addMode = false
	m.pathInput.Blur()
	m.errorMsg = ""
}

// IsAddMode returns whether we're in add mode.
func (m *ProjectManagerModel) IsAddMode() bool {
	return m.addMode
}

// GetInputValue returns the current path input value.
func (m *ProjectManagerModel) GetInputValue() string {
	return m.pathInput.Value()
}

// SetError sets an error message to display.
func (m *ProjectManagerModel) SetError(msg string) {
	m.errorMsg = msg
}

// ClearError clears the error message.
func (m *ProjectManagerModel) ClearError() {
	m.errorMsg = ""
}

// UpdateInput updates the text input with a key message.
func (m *ProjectManagerModel) UpdateInput(msg interface{}) {
	var cmd interface{}
	m.pathInput, cmd = m.pathInput.Update(msg)
	_ = cmd
}

// SelectedProject returns the currently selected project, or nil if none.
func (m *ProjectManagerModel) SelectedProject() *ProjectEntry {
	if len(m.projects) == 0 || m.selectedIndex < 0 || m.selectedIndex >= len(m.projects) {
		return nil
	}
	return &m.projects[m.selectedIndex]
}

// RemoveSelected removes the currently selected project from the list.
func (m *ProjectManagerModel) RemoveSelected() *ProjectEntry {
	if len(m.projects) == 0 || m.selectedIndex < 0 || m.selectedIndex >= len(m.projects) {
		return nil
	}
	removed := m.projects[m.selectedIndex]
	m.projects = append(m.projects[:m.selectedIndex], m.projects[m.selectedIndex+1:]...)
	if m.selectedIndex >= len(m.projects) && m.selectedIndex > 0 {
		m.selectedIndex--
	}
	return &removed
}

// AddProject adds a new project entry to the list.
func (m *ProjectManagerModel) AddProject(entry ProjectEntry) {
	m.projects = append(m.projects, entry)
}

// ActiveProjects returns entries that are currently active.
func (m *ProjectManagerModel) ActiveProjects() []ProjectEntry {
	var active []ProjectEntry
	for _, p := range m.projects {
		if p.IsActive {
			active = append(active, p)
		}
	}
	return active
}

// AllProjects returns all project entries.
func (m *ProjectManagerModel) AllProjects() []ProjectEntry {
	return m.projects
}

// View renders the project manager overlay.
func (m *ProjectManagerModel) View() string {
	if m.width == 0 {
		m.width = 80
	}
	if m.height == 0 {
		m.height = 24
	}

	t := m.theme

	// Calculate box dimensions
	boxWidth := 70
	if m.width < 80 {
		boxWidth = m.width - 10
	}
	if boxWidth < 40 {
		boxWidth = 40
	}

	var lines []string

	// Title
	titleStyle := t.Renderer.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		MarginBottom(1)
	lines = append(lines, titleStyle.Render("Project Manager"))
	lines = append(lines, "")

	if m.addMode {
		// Add mode view
		addLabel := t.Renderer.NewStyle().Foreground(t.Secondary).Render("Add project path:")
		lines = append(lines, addLabel)
		lines = append(lines, m.pathInput.View())

		if m.errorMsg != "" {
			errorStyle := t.Renderer.NewStyle().Foreground(t.Blocked) // Use Blocked color for errors
			lines = append(lines, errorStyle.Render(m.errorMsg))
		}

		lines = append(lines, "")
		footerStyle := t.Renderer.NewStyle().
			Foreground(t.Secondary).
			Italic(true)
		lines = append(lines, footerStyle.Render("enter: add • esc: cancel"))
	} else {
		// Project list view
		if len(m.projects) == 0 {
			emptyStyle := t.Renderer.NewStyle().Foreground(t.Secondary).Italic(true)
			lines = append(lines, emptyStyle.Render("No projects loaded."))
			lines = append(lines, emptyStyle.Render("Use 'a' to add a project."))
		} else {
			// Header
			headerStyle := t.Renderer.NewStyle().Foreground(t.Secondary).Underline(true)
			header := "  Name                 Path                              Issues"
			lines = append(lines, headerStyle.Render(header))

			// Project rows
			for i, proj := range m.projects {
				isCursor := i == m.selectedIndex

				nameStyle := t.Renderer.NewStyle().Foreground(t.Base.GetForeground())
				if isCursor {
					nameStyle = nameStyle.Foreground(t.Primary).Bold(true)
				}
				if !proj.IsActive {
					nameStyle = nameStyle.Foreground(t.Secondary)
				}

				cursor := "  "
				if isCursor {
					cursor = "▸ "
				}

				check := "[ ]"
				if proj.IsActive {
					check = "[x]"
				}

				// Truncate name and path for display
				name := truncateString(proj.Name, 16)
				path := truncatePathMiddle(proj.Path, 30)

				line := cursor + check + " " + padRight(name, 16) + " " + padRight(path, 32) + " " + padLeftPM(fmt.Sprintf("%d", proj.IssueCount), 5)
				lines = append(lines, nameStyle.Render(line))
			}
		}

		lines = append(lines, "")
		footerStyle := t.Renderer.NewStyle().
			Foreground(t.Secondary).
			Italic(true)
		lines = append(lines, footerStyle.Render("j/k: navigate • space: toggle • a: add • d: remove • enter: apply • esc: cancel"))
	}

	content := strings.Join(lines, "\n")

	boxStyle := t.Renderer.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1, 2).
		Width(boxWidth)
	box := boxStyle.Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

// truncatePathMiddle truncates a path in the middle, preserving start and end.
func truncatePathMiddle(path string, max int) string {
	if len(path) <= max {
		return path
	}
	if max <= 5 {
		return path[:max]
	}

	// Show first part and last part
	half := (max - 3) / 2
	return path[:half] + "..." + path[len(path)-half:]
}

// padLeftPM pads a string to the left with spaces (project manager specific).
func padLeftPM(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}

// BuildProjectEntriesFromPaths creates ProjectEntry slice from paths and issue counts.
func BuildProjectEntriesFromPaths(paths []string, prefixes []string, issueCounts map[string]int) []ProjectEntry {
	var entries []ProjectEntry
	for i, path := range paths {
		var prefix string
		if i < len(prefixes) {
			prefix = prefixes[i]
		}
		count := 0
		if issueCounts != nil {
			count = issueCounts[prefix]
		}
		entries = append(entries, ProjectEntry{
			Name:       filepath.Base(path),
			Path:       path,
			Prefix:     prefix,
			IssueCount: count,
			IsActive:   true,
		})
	}
	return entries
}
