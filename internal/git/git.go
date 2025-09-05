package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CheckGitPresence checks if git is available and returns version
func CheckGitPresence() (string, error) {
	cmd := exec.Command("git", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git not found: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// IsGitRepo checks if the current directory is a git repository
func IsGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	return isDir(gitDir)
}

// FindGitRoot finds the root of the git repository containing the given path
func FindGitRoot(path string) (string, error) {
	current := path
	for {
		if IsGitRepo(current) {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("not in a git repository")
		}
		current = parent
	}
}

// GetRemoteURL gets the origin remote URL
func GetRemoteURL(repoPath string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// SetRemoteURL sets the origin remote URL
func SetRemoteURL(repoPath, url string) error {
	cmd := exec.Command("git", "remote", "set-url", "origin", url)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set remote URL: %w", err)
	}
	return nil
}

// GetLocalConfig gets a local git config value
func GetLocalConfig(repoPath, key string) (string, error) {
	cmd := exec.Command("git", "config", "--local", key)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get local config %s: %w", key, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// SetLocalConfig sets a local git config value
func SetLocalConfig(repoPath, key, value string) error {
	cmd := exec.Command("git", "config", "--local", key, value)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set local config %s: %w", key, err)
	}
	return nil
}

// UnsetLocalConfig unsets a local git config value
func UnsetLocalConfig(repoPath, key string) error {
	cmd := exec.Command("git", "config", "--local", "--unset", key)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		// Ignore error if key doesn't exist
		return nil
	}
	return nil
}

// CloneRepository clones a repository
func CloneRepository(url, destPath, branch string) error {
	args := []string{"clone"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, url, destPath)

	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	return nil
}

// InstallHooks installs pre-commit and pre-push hooks
func InstallHooks(repoPath string) error {
	hookDir := filepath.Join(repoPath, ".git", "hooks")

	// Install pre-commit hook
	preCommitHook := `#!/bin/sh
# Git Workspace Guard - Pre-commit Hook

# Get current user email
CURRENT_EMAIL=$(git config user.email)

# Get workspace from remote URL
REMOTE_URL=$(git remote get-url origin 2>/dev/null)
if [ -z "$REMOTE_URL" ]; then
    echo "Warning: No origin remote found"
    exit 0
fi

# Extract host from SSH URL (e.g., git@github-work:org/repo.git -> github-work)
HOST=$(echo "$REMOTE_URL" | sed -n 's/git@\([^:]*\):.*/\1/p')

if [ -z "$HOST" ]; then
    echo "Warning: Could not extract host from remote URL"
    exit 0
fi

# Check if this is a gitws managed workspace
if echo "$HOST" | grep -q "gws\|gitws"; then
    echo "✓ Git workspace guard: Using managed workspace"
    exit 0
fi

# For non-managed workspaces, just warn
echo "⚠️  Git workspace guard: Using unmanaged workspace ($HOST)"
echo "   Current email: $CURRENT_EMAIL"
echo "   Consider using 'gitws init' to set up workspace isolation"
exit 0
`

	preCommitPath := filepath.Join(hookDir, "pre-commit")
	if err := os.WriteFile(preCommitPath, []byte(preCommitHook), 0755); err != nil {
		return fmt.Errorf("failed to write pre-commit hook: %w", err)
	}

	// Install pre-push hook
	prePushHook := `#!/bin/sh
# Git Workspace Guard - Pre-push Hook

# Get current user email
CURRENT_EMAIL=$(git config user.email)

# Get workspace from remote URL
REMOTE_URL=$(git remote get-url origin 2>/dev/null)
if [ -z "$REMOTE_URL" ]; then
    echo "Warning: No origin remote found"
    exit 0
fi

# Extract host from SSH URL
HOST=$(echo "$REMOTE_URL" | sed -n 's/git@\([^:]*\):.*/\1/p')

if [ -z "$HOST" ]; then
    echo "Warning: Could not extract host from remote URL"
    exit 0
fi

# Check if this is a gitws managed workspace
if echo "$HOST" | grep -q "gws\|gitws"; then
    echo "✓ Git workspace guard: Using managed workspace"
    exit 0
fi

# For non-managed workspaces, just warn
echo "⚠️  Git workspace guard: Using unmanaged workspace ($HOST)"
echo "   Current email: $CURRENT_EMAIL"
echo "   Consider using 'gitws init' to set up workspace isolation"
exit 0
`

	prePushPath := filepath.Join(hookDir, "pre-push")
	if err := os.WriteFile(prePushPath, []byte(prePushHook), 0755); err != nil {
		return fmt.Errorf("failed to write pre-push hook: %w", err)
	}

	return nil
}

// CheckHooksInstalled checks if hooks are installed
func CheckHooksInstalled(repoPath string) (bool, error) {
	hookDir := filepath.Join(repoPath, ".git", "hooks")

	preCommitPath := filepath.Join(hookDir, "pre-commit")
	prePushPath := filepath.Join(hookDir, "pre-push")

	preCommitExists := isFile(preCommitPath)
	prePushExists := isFile(prePushPath)

	return preCommitExists && prePushExists, nil
}

// GetSigningStatus gets the current signing configuration
func GetSigningStatus(repoPath string) (enabled bool, method string, key string, err error) {
	// Check if signing is enabled
	signCommit, err := GetLocalConfig(repoPath, "commit.gpgsign")
	if err != nil {
		// Check global config
		cmd := exec.Command("git", "config", "--global", "commit.gpgsign")
		cmd.Dir = repoPath
		output, err := cmd.Output()
		if err != nil {
			return false, "", "", nil // Signing not configured
		}
		signCommit = strings.TrimSpace(string(output))
	}

	enabled = signCommit == "true"
	if !enabled {
		return false, "", "", nil
	}

	// Get signing method
	gpgFormat, err := GetLocalConfig(repoPath, "gpg.format")
	if err != nil {
		// Check global config
		cmd := exec.Command("git", "config", "--global", "gpg.format")
		cmd.Dir = repoPath
		output, err := cmd.Output()
		if err != nil {
			method = "gpg" // Default
		} else {
			method = strings.TrimSpace(string(output))
		}
	} else {
		method = gpgFormat
	}

	// Get signing key
	signingKey, err := GetLocalConfig(repoPath, "user.signingkey")
	if err != nil {
		// Check global config
		cmd := exec.Command("git", "config", "--global", "user.signingkey")
		cmd.Dir = repoPath
		output, err := cmd.Output()
		if err != nil {
			key = ""
		} else {
			key = strings.TrimSpace(string(output))
		}
	} else {
		key = signingKey
	}

	return enabled, method, key, nil
}

// Helper functions
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
