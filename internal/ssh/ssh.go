package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gitworkspaces/gitws/internal/fsutil"
	"github.com/gitworkspaces/gitws/internal/workspace"
)

// EnsureKey creates an SSH key for the workspace if it doesn't exist
func EnsureKey(workspaceName, email string) (privPath, pubPath string, created bool, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", false, fmt.Errorf("failed to get home directory: %w", err)
	}

	keyName := fmt.Sprintf("id_ed25519_gws_%s", workspaceName)
	privPath = filepath.Join(home, ".ssh", keyName)
	pubPath = privPath + ".pub"

	// Check if key already exists
	if fsutil.FileExists(privPath) {
		return privPath, pubPath, false, nil
	}

	// Ensure .ssh directory exists
	sshDir := filepath.Join(home, ".ssh")
	if err := fsutil.EnsureDir(sshDir); err != nil {
		return "", "", false, fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	// Generate SSH key
	comment := fmt.Sprintf("%s gws-%s", email, workspaceName)
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", comment, "-f", privPath, "-N", "")

	if err := cmd.Run(); err != nil {
		return "", "", false, fmt.Errorf("failed to generate SSH key: %w", err)
	}

	// Set proper permissions
	if err := os.Chmod(privPath, 0600); err != nil {
		return "", "", false, fmt.Errorf("failed to set key permissions: %w", err)
	}

	return privPath, pubPath, true, nil
}

// UpsertSSHConfigBlock updates the SSH config with a managed block for the workspace
func UpsertSSHConfigBlock(workspaceName, alias, hostName, keyPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".ssh", "config")

	// Read existing config
	var content string
	if fsutil.FileExists(configPath) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read SSH config: %w", err)
		}
		content = string(data)
	}

	// Create backup
	if err := fsutil.CreateBackup(configPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Build new block
	startMarker := workspace.StartMarker(workspaceName)
	endMarker := workspace.EndMarker(workspaceName)

	newBlock := fmt.Sprintf(`%s
Host %s
  HostName %s
  User git
  IdentityFile %s
  IdentitiesOnly yes
%s`, startMarker, alias, hostName, keyPath, endMarker)

	// Replace content between markers
	newContent, _ := fsutil.ReplaceBetweenMarkers(content, startMarker, endMarker, newBlock)

	// Write updated config
	if err := fsutil.AtomicWrite(configPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	return nil
}

// GetPublicKey reads the public key content
func GetPublicKey(pubPath string) (string, error) {
	data, err := os.ReadFile(pubPath)
	if err != nil {
		return "", fmt.Errorf("failed to read public key: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// TestSSHConnection tests SSH connection to a host
func TestSSHConnection(alias string) error {
	cmd := exec.Command("ssh", "-T", alias, "-o", "ConnectTimeout=10", "-o", "BatchMode=yes")
	cmd.Stdout = nil
	cmd.Stderr = nil

	_ = cmd.Run()
	// SSH returns exit code 1 for successful connection to Git servers
	// Exit code 255 indicates connection failure
	if cmd.ProcessState.ExitCode() == 255 {
		return fmt.Errorf("SSH connection to %s failed", alias)
	}

	return nil
}

// RemoveSSHConfigBlock removes the managed block for a workspace
func RemoveSSHConfigBlock(workspaceName string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".ssh", "config")

	if !fsutil.FileExists(configPath) {
		return nil // No config file to modify
	}

	// Read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read SSH config: %w", err)
	}
	content := string(data)

	// Create backup
	if err := fsutil.CreateBackup(configPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Remove content between markers
	startMarker := workspace.StartMarker(workspaceName)
	endMarker := workspace.EndMarker(workspaceName)

	startIdx := strings.Index(content, startMarker)
	if startIdx == -1 {
		return nil // Block not found
	}

	endIdx := strings.Index(content[startIdx:], endMarker)
	if endIdx == -1 {
		return nil // End marker not found
	}

	endIdx += startIdx + len(endMarker)

	// Remove content between markers
	before := content[:startIdx]
	after := content[endIdx:]
	newContent := before + after

	// Write updated config
	if err := fsutil.AtomicWrite(configPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	return nil
}
