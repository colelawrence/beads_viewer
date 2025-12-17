// Package config provides user-level configuration for bv.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProjectsFileName is the name of the projects config file.
const ProjectsFileName = "projects.yaml"

// ProjectsConfig holds the user's saved project list.
type ProjectsConfig struct {
	// Projects is the list of saved projects.
	Projects []ProjectEntry `yaml:"projects"`
}

// ProjectEntry represents a single project in the saved config.
type ProjectEntry struct {
	// Name is an optional display name for the project.
	Name string `yaml:"name,omitempty"`
	// Path is the absolute path to the project directory.
	Path string `yaml:"path"`
	// Enabled indicates whether this project should be loaded (default: true).
	Enabled *bool `yaml:"enabled,omitempty"`
}

// IsEnabled returns whether the project is enabled.
func (p *ProjectEntry) IsEnabled() bool {
	if p.Enabled == nil {
		return true
	}
	return *p.Enabled
}

// DefaultConfigDir returns the bv config directory.
// Uses XDG_CONFIG_HOME if set, otherwise ~/.config/bv.
func DefaultConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "bv")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "bv")
}

// ProjectsConfigPath returns the full path to the projects config file.
func ProjectsConfigPath() string {
	return filepath.Join(DefaultConfigDir(), ProjectsFileName)
}

// LoadProjects loads the projects config from the default location.
// Returns an empty config if the file doesn't exist.
func LoadProjects() (*ProjectsConfig, error) {
	return LoadProjectsFrom(ProjectsConfigPath())
}

// LoadProjectsFrom loads the projects config from a specific path.
func LoadProjectsFrom(path string) (*ProjectsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ProjectsConfig{}, nil
		}
		return nil, err
	}

	var config ProjectsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// SaveProjects saves the projects config to the default location.
func SaveProjects(config *ProjectsConfig) error {
	return SaveProjectsTo(config, ProjectsConfigPath())
}

// SaveProjectsTo saves the projects config to a specific path.
func SaveProjectsTo(config *ProjectsConfig, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// ClearProjects removes the projects config file.
func ClearProjects() error {
	path := ProjectsConfigPath()
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// AddProject adds a project to the config if it doesn't already exist.
// Returns true if the project was added, false if it already existed.
func (c *ProjectsConfig) AddProject(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	// Check if already exists
	for _, p := range c.Projects {
		if p.Path == absPath {
			return false
		}
	}

	c.Projects = append(c.Projects, ProjectEntry{
		Name: filepath.Base(absPath),
		Path: absPath,
	})
	return true
}

// RemoveProject removes a project from the config by path.
// Returns true if the project was removed, false if it wasn't found.
func (c *ProjectsConfig) RemoveProject(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	for i, p := range c.Projects {
		if p.Path == absPath {
			c.Projects = append(c.Projects[:i], c.Projects[i+1:]...)
			return true
		}
	}
	return false
}

// EnabledPaths returns the paths of all enabled projects.
func (c *ProjectsConfig) EnabledPaths() []string {
	var paths []string
	for _, p := range c.Projects {
		if p.IsEnabled() {
			paths = append(paths, p.Path)
		}
	}
	return paths
}
