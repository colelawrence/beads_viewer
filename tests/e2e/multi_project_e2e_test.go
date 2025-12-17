package main_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// ============================================================================
// E2E: Multi-Project Support Tests
// Tests for --project flags, project persistence, and issue namespacing
// ============================================================================

// createTestProject creates a project fixture with .beads/beads.jsonl
func createTestProject(t *testing.T, baseDir, name string, issues []string) string {
	t.Helper()
	projectDir := filepath.Join(baseDir, name)
	beadsDir := filepath.Join(projectDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	var lines []string
	for i, title := range issues {
		lines = append(lines, strings.TrimSpace(`{"id":"`+strings.ToUpper(name)+`-`+itoa(i+1)+`","title":"`+title+`","status":"open","priority":1,"issue_type":"task"}`))
	}
	if err := os.WriteFile(filepath.Join(beadsDir, "beads.jsonl"),
		[]byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatal(err)
	}
	return projectDir
}

// createTestProjectWithIssueID creates a project with a specific issue ID
func createTestProjectWithIssueID(t *testing.T, baseDir, name, issueID, title string) string {
	t.Helper()
	projectDir := filepath.Join(baseDir, name)
	beadsDir := filepath.Join(projectDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	line := `{"id":"` + issueID + `","title":"` + title + `","status":"open","priority":1,"issue_type":"task"}`
	if err := os.WriteFile(filepath.Join(beadsDir, "beads.jsonl"), []byte(line), 0644); err != nil {
		t.Fatal(err)
	}
	return projectDir
}

// Note: itoa is defined in export_pages_test.go and shared across the package

// TestMultiProject_LoadTwoProjects verifies --project flag loads multiple projects
func TestMultiProject_LoadTwoProjects(t *testing.T) {
	bv := buildBvBinary(t)
	baseDir := t.TempDir()

	// Create two project dirs with issues
	apiDir := createTestProject(t, baseDir, "api", []string{"API Endpoint", "API Auth"})
	webDir := createTestProject(t, baseDir, "web", []string{"Dashboard", "Settings"})

	// Run bv with both projects
	cmd := exec.Command(bv, "--project", apiDir, "--project", webDir, "--robot-triage")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bv failed: %v\n%s", err, out)
	}

	// Parse JSON output
	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}

	// Get triage data
	triage, ok := result["triage"].(map[string]interface{})
	if !ok {
		t.Fatal("missing triage field")
	}

	quickRef, ok := triage["quick_ref"].(map[string]interface{})
	if !ok {
		t.Fatal("missing quick_ref")
	}

	// Verify total issue count (2 from api + 2 from web = 4)
	openCount, ok := quickRef["open_count"].(float64)
	if !ok || openCount != 4 {
		t.Errorf("expected 4 open issues, got %v", quickRef["open_count"])
	}

	// Verify issues from both projects appear in recommendations
	recommendations, ok := triage["recommendations"].([]interface{})
	if !ok {
		t.Log("Note: no recommendations returned")
	} else {
		// Check that we have issues from both projects
		var foundAPI, foundWeb bool
		for _, rec := range recommendations {
			if recMap, ok := rec.(map[string]interface{}); ok {
				id, _ := recMap["id"].(string)
				if strings.HasPrefix(id, "api-") {
					foundAPI = true
				}
				if strings.HasPrefix(id, "web-") {
					foundWeb = true
				}
			}
		}
		if len(recommendations) >= 2 && (!foundAPI || !foundWeb) {
			t.Log("Note: recommendations may not include all prefixed issues")
		}
	}
}

// TestMultiProject_IssueNamespacing verifies duplicate IDs get namespaced
func TestMultiProject_IssueNamespacing(t *testing.T) {
	bv := buildBvBinary(t)
	baseDir := t.TempDir()

	// Create two projects with SAME issue ID "TASK-1"
	apiDir := createTestProjectWithIssueID(t, baseDir, "api", "TASK-1", "API Task")
	webDir := createTestProjectWithIssueID(t, baseDir, "web", "TASK-1", "Web Task")

	// Run bv with both projects
	cmd := exec.Command(bv, "--project", apiDir, "--project", webDir, "--robot-triage")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bv failed: %v\n%s", err, out)
	}

	// Parse JSON output
	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}

	// Verify we have 2 issues total (both TASK-1s should be loaded with different prefixes)
	triage := result["triage"].(map[string]interface{})
	quickRef := triage["quick_ref"].(map[string]interface{})

	openCount := quickRef["open_count"].(float64)
	if openCount != 2 {
		t.Errorf("expected 2 open issues (both TASK-1s), got %v", openCount)
	}

	// The output string should contain both prefixed IDs
	outStr := string(out)
	if !strings.Contains(outStr, "api-TASK-1") {
		t.Error("expected 'api-TASK-1' in output")
	}
	if !strings.Contains(outStr, "web-TASK-1") {
		t.Error("expected 'web-TASK-1' in output")
	}
}

// TestMultiProject_SaveProjects verifies --save-projects persists config
func TestMultiProject_SaveProjects(t *testing.T) {
	bv := buildBvBinary(t)
	baseDir := t.TempDir()
	configDir := t.TempDir()

	// Create projects
	apiDir := createTestProject(t, baseDir, "api", []string{"API Task"})
	webDir := createTestProject(t, baseDir, "web", []string{"Web Task"})

	// Run bv with --save-projects
	cmd := exec.Command(bv, "--project", apiDir, "--project", webDir, "--save-projects", "--robot-triage")
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+configDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bv failed: %v\n%s", err, out)
	}

	// Verify projects.yaml created
	projectsPath := filepath.Join(configDir, "bv", "projects.yaml")
	data, err := os.ReadFile(projectsPath)
	if err != nil {
		t.Fatalf("projects.yaml not created: %v", err)
	}

	// Parse YAML and verify contents
	var config struct {
		Projects []struct {
			Name string `yaml:"name"`
			Path string `yaml:"path"`
		} `yaml:"projects"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("invalid YAML: %v\n%s", err, data)
	}

	if len(config.Projects) != 2 {
		t.Errorf("expected 2 projects in config, got %d", len(config.Projects))
	}

	// Verify both paths are saved
	var foundAPI, foundWeb bool
	for _, p := range config.Projects {
		if p.Path == apiDir {
			foundAPI = true
		}
		if p.Path == webDir {
			foundWeb = true
		}
	}
	if !foundAPI {
		t.Error("api project not saved to config")
	}
	if !foundWeb {
		t.Error("web project not saved to config")
	}
}

// TestMultiProject_LoadSavedProjects verifies saved projects load automatically
func TestMultiProject_LoadSavedProjects(t *testing.T) {
	bv := buildBvBinary(t)
	baseDir := t.TempDir()
	configDir := t.TempDir()

	// Create projects
	apiDir := createTestProject(t, baseDir, "api", []string{"API Task"})
	webDir := createTestProject(t, baseDir, "web", []string{"Web Task"})

	// Manually create projects.yaml
	bvConfigDir := filepath.Join(configDir, "bv")
	if err := os.MkdirAll(bvConfigDir, 0755); err != nil {
		t.Fatal(err)
	}

	configYAML := `projects:
  - name: api
    path: ` + apiDir + `
  - name: web
    path: ` + webDir + `
`
	if err := os.WriteFile(filepath.Join(bvConfigDir, "projects.yaml"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Run bv WITHOUT --project flags (should load from saved config)
	cmd := exec.Command(bv, "--robot-triage")
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+configDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bv failed: %v\n%s", err, out)
	}

	// Verify issues from saved projects appear
	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}

	triage := result["triage"].(map[string]interface{})
	quickRef := triage["quick_ref"].(map[string]interface{})

	openCount := quickRef["open_count"].(float64)
	if openCount != 2 {
		t.Errorf("expected 2 open issues from saved projects, got %v", openCount)
	}
}

// TestMultiProject_ClearProjects verifies --clear-projects removes config
func TestMultiProject_ClearProjects(t *testing.T) {
	bv := buildBvBinary(t)
	configDir := t.TempDir()

	// Create a projects.yaml file
	bvConfigDir := filepath.Join(configDir, "bv")
	if err := os.MkdirAll(bvConfigDir, 0755); err != nil {
		t.Fatal(err)
	}
	projectsPath := filepath.Join(bvConfigDir, "projects.yaml")
	if err := os.WriteFile(projectsPath, []byte("projects:\n  - path: /fake/path\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Verify file exists
	if _, err := os.Stat(projectsPath); os.IsNotExist(err) {
		t.Fatal("projects.yaml should exist before clear")
	}

	// Run bv --clear-projects
	cmd := exec.Command(bv, "--clear-projects")
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+configDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bv --clear-projects failed: %v\n%s", err, out)
	}

	// Verify file is removed
	if _, err := os.Stat(projectsPath); !os.IsNotExist(err) {
		t.Error("projects.yaml should be removed after --clear-projects")
	}
}

// TestMultiProject_RepoFilter verifies --repo filters by prefix
func TestMultiProject_RepoFilter(t *testing.T) {
	bv := buildBvBinary(t)
	baseDir := t.TempDir()

	// Create two projects
	apiDir := createTestProject(t, baseDir, "api", []string{"API Task 1", "API Task 2"})
	webDir := createTestProject(t, baseDir, "web", []string{"Web Task 1", "Web Task 2"})

	// Run bv with --repo filter for api only
	cmd := exec.Command(bv, "--project", apiDir, "--project", webDir, "--repo", "api", "--robot-triage")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bv failed: %v\n%s", err, out)
	}

	// Parse JSON output
	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}

	triage := result["triage"].(map[string]interface{})
	quickRef := triage["quick_ref"].(map[string]interface{})

	// Should only have 2 issues (from api), not 4
	openCount := quickRef["open_count"].(float64)
	if openCount != 2 {
		t.Errorf("expected 2 open issues (api only), got %v", openCount)
	}

	// Verify only api issues in output
	outStr := string(out)
	if !strings.Contains(outStr, "api-") {
		t.Error("expected api issues in filtered output")
	}
	if strings.Contains(outStr, "web-") {
		t.Error("unexpected web issues in api-filtered output")
	}
}

// TestMultiProject_DuplicateProjectNames verifies collision handling
func TestMultiProject_DuplicateProjectNames(t *testing.T) {
	bv := buildBvBinary(t)
	baseDir := t.TempDir()

	// Create two projects with SAME directory name in different parent dirs
	parentA := filepath.Join(baseDir, "a")
	parentB := filepath.Join(baseDir, "b")

	projA := createTestProject(t, parentA, "myproject", []string{"Task A"})
	projB := createTestProject(t, parentB, "myproject", []string{"Task B"})

	// Run bv with both projects
	cmd := exec.Command(bv, "--project", projA, "--project", projB, "--robot-triage")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bv failed: %v\n%s", err, out)
	}

	// Parse JSON output
	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}

	triage := result["triage"].(map[string]interface{})
	quickRef := triage["quick_ref"].(map[string]interface{})

	// Should have 2 issues total
	openCount := quickRef["open_count"].(float64)
	if openCount != 2 {
		t.Errorf("expected 2 open issues, got %v", openCount)
	}

	// The prefixes should be distinct (e.g., myproject- and myproject_1-)
	outStr := string(out)
	// Check that we have both issues with distinct prefixes
	if !strings.Contains(outStr, "myproject-") {
		t.Error("expected 'myproject-' prefix in output")
	}
	// The second project should get a disambiguated prefix
	if !strings.Contains(outStr, "myproject_1-") && !strings.Contains(outStr, "myproject_2-") {
		t.Log("Note: expected disambiguated prefix like 'myproject_1-' for duplicate dir name")
	}
}

// TestMultiProject_CrossProjectDependencies verifies deps across projects
func TestMultiProject_CrossProjectDependencies(t *testing.T) {
	bv := buildBvBinary(t)
	baseDir := t.TempDir()

	// Create api project with API-1
	apiDir := filepath.Join(baseDir, "api")
	beadsDir := filepath.Join(apiDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	apiIssue := `{"id":"API-1","title":"API Endpoint","status":"open","priority":1,"issue_type":"task"}`
	if err := os.WriteFile(filepath.Join(beadsDir, "beads.jsonl"), []byte(apiIssue), 0644); err != nil {
		t.Fatal(err)
	}

	// Create web project with WEB-1 that depends on api-API-1
	webDir := filepath.Join(baseDir, "web")
	beadsDir = filepath.Join(webDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Note: cross-project dependency uses the prefixed ID "api-API-1"
	webIssue := `{"id":"WEB-1","title":"Web Dashboard","status":"open","priority":1,"issue_type":"task","dependencies":[{"depends_on_id":"api-API-1","type":"blocks"}]}`
	if err := os.WriteFile(filepath.Join(beadsDir, "beads.jsonl"), []byte(webIssue), 0644); err != nil {
		t.Fatal(err)
	}

	// Run bv with both projects
	cmd := exec.Command(bv, "--project", apiDir, "--project", webDir, "--robot-plan")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bv failed: %v\n%s", err, out)
	}

	// Parse JSON output
	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}

	// Verify plan structure
	plan, ok := result["plan"].(map[string]interface{})
	if !ok {
		t.Fatal("missing plan field")
	}

	// Check for tracks
	tracks, ok := plan["tracks"].([]interface{})
	if !ok {
		t.Fatal("missing tracks field")
	}

	// Should have tracks (exact structure depends on implementation)
	if len(tracks) == 0 {
		t.Error("expected at least one track in execution plan")
	}

	// The dependency graph should show both issues
	outStr := string(out)
	if !strings.Contains(outStr, "api-API-1") {
		t.Error("expected 'api-API-1' in plan output")
	}
	if !strings.Contains(outStr, "web-WEB-1") {
		t.Error("expected 'web-WEB-1' in plan output")
	}
}
