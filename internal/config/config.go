package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Workspace represents a git workspace configuration
type Workspace struct {
	Email    string `yaml:"email"`
	Provider string `yaml:"provider"`  // "github"|"gitlab"|"bitbucket"|"" if custom
	HostName string `yaml:"host_name"` // fqdn
	SSHAlias string `yaml:"ssh_alias"`
	SSHKey   string `yaml:"ssh_key"`
	Root     string `yaml:"root"`
	Signing  string `yaml:"signing"` // "none"|"ssh"|"gpg"
	Name     string `yaml:"name"`
}

// File represents the complete configuration file
type File struct {
	Workspaces map[string]Workspace `yaml:"workspaces"`
}

// ConfigDir returns the configuration directory path
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".gws"), nil
}

// ConfigPath returns the path to the configuration file
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load loads the configuration from disk
func Load() (*File, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{Workspaces: make(map[string]Workspace)}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config File
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.Workspaces == nil {
		config.Workspaces = make(map[string]Workspace)
	}

	return &config, nil
}

// Save saves the configuration to disk
func (f *File) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(f)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetWorkspace returns a workspace by name
func (f *File) GetWorkspace(name string) (Workspace, bool) {
	ws, exists := f.Workspaces[name]
	return ws, exists
}

// SetWorkspace sets a workspace configuration
func (f *File) SetWorkspace(name string, ws Workspace) {
	if f.Workspaces == nil {
		f.Workspaces = make(map[string]Workspace)
	}
	f.Workspaces[name] = ws
}

// DeleteWorkspace removes a workspace
func (f *File) DeleteWorkspace(name string) {
	delete(f.Workspaces, name)
}

// ListWorkspaces returns all workspace names
func (f *File) ListWorkspaces() []string {
	var names []string
	for name := range f.Workspaces {
		names = append(names, name)
	}
	return names
}
