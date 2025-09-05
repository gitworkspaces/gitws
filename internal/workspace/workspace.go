package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ProviderHosts maps provider names to their hostnames
var ProviderHosts = map[string]string{
	"github":    "github.com",
	"gitlab":    "gitlab.com",
	"bitbucket": "bitbucket.org",
}

// BuildSSHAlias creates an SSH alias from provider/host and workspace name
func BuildSSHAlias(providerOrHost, workspace string) string {
	// Use provider hostname if it's a known provider
	host := providerOrHost
	if providerHost, exists := ProviderHosts[providerOrHost]; exists {
		host = providerHost
	}

	// Create alias: <host>-<workspace>
	alias := fmt.Sprintf("%s-%s", host, workspace)

	// Slugify: lowercase, replace non-alphanumeric with dashes
	alias = strings.ToLower(alias)
	alias = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(alias, "-")

	// Remove multiple consecutive dashes
	alias = regexp.MustCompile(`-+`).ReplaceAllString(alias, "-")

	// Remove leading/trailing dashes
	alias = strings.Trim(alias, "-")

	// Truncate to 63 characters (SSH hostname limit)
	if len(alias) > 63 {
		alias = alias[:63]
		// Ensure we don't end with a dash
		alias = strings.TrimSuffix(alias, "-")
	}

	return alias
}

// ExpandPath expands ~ in paths to the user's home directory
func ExpandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

// DefaultRoot returns the default root path for a workspace
func DefaultRoot(workspace string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, "code", workspace), nil
}

// GitConfigPath returns the path to a workspace's git config file
func GitConfigPath(workspace string) (string, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "gitconfig", workspace), nil
}

// ConfigDir returns the configuration directory path
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".gws"), nil
}

// BuildIncludeIfCondition creates the gitdir condition for includeIf
func BuildIncludeIfCondition(root string) (string, error) {
	expandedRoot, err := ExpandPath(root)
	if err != nil {
		return "", err
	}

	// Ensure path ends with / for gitdir matching
	if !strings.HasSuffix(expandedRoot, "/") {
		expandedRoot += "/"
	}

	return fmt.Sprintf("gitdir:%s", expandedRoot), nil
}

// StartMarker returns the start marker for managed blocks
func StartMarker(workspace string) string {
	return fmt.Sprintf("# >>> gws %s >>> DO NOT EDIT", workspace)
}

// EndMarker returns the end marker for managed blocks
func EndMarker(workspace string) string {
	return fmt.Sprintf("# <<< gws %s <<<", workspace)
}

// IncludeIfStartMarker returns the start marker for includeIf blocks
func IncludeIfStartMarker() string {
	return "# >>> gws includeIf >>> DO NOT EDIT"
}

// IncludeIfEndMarker returns the end marker for includeIf blocks
func IncludeIfEndMarker() string {
	return "# <<< gws includeIf <<<"
}
