package gws

import (
	"fmt"
	"os"
	"time"

	"github.com/gitworkspaces/gitws/internal/config"
	"github.com/gitworkspaces/gitws/internal/prompt"
	"github.com/gitworkspaces/gitws/internal/ssh"
	"github.com/spf13/cobra"
)

// rotateCmd represents the rotate command
var rotateCmd = &cobra.Command{
	Use:   "rotate <workspace>",
	Short: "Rotate SSH keys for a workspace",
	Long: `Generate new SSH keys for a workspace and update configuration.

This command will:
- Generate a new SSH key pair with timestamp
- Backup the old key with timestamp
- Update SSH configuration
- Display the new public key

Examples:
  gitws rotate work
  gitws rotate personal`,
	Args: cobra.ExactArgs(1),
	RunE: runRotate,
}

func init() {
	rootCmd.AddCommand(rotateCmd)
}

func runRotate(cmd *cobra.Command, args []string) error {
	workspaceName := args[0]

	// Load workspace config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ws, exists := cfg.GetWorkspace(workspaceName)
	if !exists {
		return fmt.Errorf("workspace %q not found", workspaceName)
	}

	// Confirm rotation
	confirmed, err := prompt.Confirm(fmt.Sprintf("Rotate SSH keys for workspace '%s'? This will generate new keys and backup the old ones.", workspaceName))
	if err != nil {
		return fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !confirmed {
		fmt.Println("Key rotation cancelled.")
		return nil
	}

	// Backup existing key
	if err := backupExistingKey(ws.SSHKey); err != nil {
		return fmt.Errorf("failed to backup existing key: %w", err)
	}

	// Generate new key
	privPath, pubPath, _, err := ssh.EnsureKey(workspaceName, ws.Email)
	if err != nil {
		return fmt.Errorf("failed to generate new key: %w", err)
	}

	// Update SSH config with new key
	if err := ssh.UpsertSSHConfigBlock(workspaceName, ws.SSHAlias, ws.HostName, privPath); err != nil {
		return fmt.Errorf("failed to update SSH config: %w", err)
	}

	// Update workspace config
	ws.SSHKey = privPath
	cfg.SetWorkspace(workspaceName, ws)
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Get new public key
	publicKey, err := ssh.GetPublicKey(pubPath)
	if err != nil {
		return fmt.Errorf("failed to read new public key: %w", err)
	}

	// Show summary
	summary := prompt.SummaryData{
		Title: fmt.Sprintf("✓ SSH keys rotated for workspace '%s'", workspaceName),
		Items: []prompt.SummaryItem{
			{Label: "New Private Key", Value: privPath, Icon: "🔑"},
			{Label: "New Public Key", Value: pubPath, Icon: "🔓"},
			{Label: "SSH Alias", Value: ws.SSHAlias, Icon: "🔗"},
			{Label: "Host", Value: ws.HostName, Icon: "🌐"},
		},
		PublicKey: publicKey,
		NextSteps: []string{
			fmt.Sprintf("Add the new public key to your %s account", ws.HostName),
			"Remove the old public key from your account",
			"Test SSH connection: ssh -T " + ws.SSHAlias,
		},
	}

	return prompt.ShowSummary(summary)
}

func backupExistingKey(keyPath string) error {
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return nil // No existing key to backup
	}

	// Create timestamped backup
	timestamp := time.Now().Format("20060102150405")
	backupPath := keyPath + ".old-" + timestamp

	// Copy private key
	if err := copyFile(keyPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup private key: %w", err)
	}

	// Copy public key if it exists
	pubPath := keyPath + ".pub"
	if _, err := os.Stat(pubPath); err == nil {
		backupPubPath := pubPath + ".old-" + timestamp
		if err := copyFile(pubPath, backupPubPath); err != nil {
			return fmt.Errorf("failed to backup public key: %w", err)
		}
	}

	fmt.Printf("✓ Backed up existing keys with timestamp: %s\n", timestamp)
	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = dstFile.ReadFrom(srcFile)
	return err
}
