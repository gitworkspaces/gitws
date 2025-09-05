package gws

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gitworkspaces/gitws/internal/config"
	"github.com/gitworkspaces/gitws/internal/git"
	"github.com/gitworkspaces/gitws/internal/prompt"
	"github.com/gitworkspaces/gitws/internal/rewrite"
	"github.com/spf13/cobra"
)

var (
	cloneBranch string
)

// cloneCmd represents the clone command
var cloneCmd = &cobra.Command{
	Use:   "clone <workspace> <url-or-org/repo>",
	Short: "Clone a repository into a workspace",
	Long: `Clone a repository using workspace-specific SSH configuration.

This command will:
- Rewrite the URL to use the workspace SSH alias
- Clone into the workspace root directory
- Set up proper Git configuration for the repository

Examples:
  gitws clone work microsoft/vscode
  gitws clone personal myorg/myrepo --branch main
  gitws clone work https://github.com/microsoft/vscode.git`,
	Args: cobra.ExactArgs(2),
	RunE: runClone,
}

func init() {
	rootCmd.AddCommand(cloneCmd)

	cloneCmd.Flags().StringVarP(&cloneBranch, "branch", "b", "", "Branch to clone")
}

func runClone(cmd *cobra.Command, args []string) error {
	workspaceName := args[0]
	urlOrRepo := args[1]

	// Load workspace config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ws, exists := cfg.GetWorkspace(workspaceName)
	if !exists {
		return fmt.Errorf("workspace %q not found. Run 'gitws init %s' first", workspaceName, workspaceName)
	}

	// Rewrite URL
	org, repo, sshURL, err := rewrite.RewriteURL(urlOrRepo, ws.SSHAlias)
	if err != nil {
		return fmt.Errorf("failed to rewrite URL: %w", err)
	}

	// Build destination path
	destPath := filepath.Join(ws.Root, org, repo)

	// Ensure parent directory exists
	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Check if destination already exists
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("destination %s already exists", destPath)
	}

	// Clone repository
	if err := git.CloneRepository(sshURL, destPath, cloneBranch); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Set up repository configuration
	if err := setupRepositoryConfig(destPath, ws); err != nil {
		return fmt.Errorf("failed to setup repository config: %w", err)
	}

	// Show summary
	summary := prompt.SummaryData{
		Title: "✓ Repository cloned successfully",
		Items: []prompt.SummaryItem{
			{Label: "Workspace", Value: workspaceName, Icon: "📁"},
			{Label: "Repository", Value: fmt.Sprintf("%s/%s", org, repo), Icon: "📦"},
			{Label: "Destination", Value: destPath, Icon: "📍"},
			{Label: "SSH URL", Value: sshURL, Icon: "🔗"},
			{Label: "Branch", Value: getBranchDisplay(cloneBranch), Icon: "🌿"},
		},
		NextSteps: []string{
			fmt.Sprintf("cd %s", destPath),
			"Run 'gitws status' to verify configuration",
			"Start working with your isolated Git identity!",
		},
	}

	return prompt.ShowSummary(summary)
}

func setupRepositoryConfig(repoPath string, ws config.Workspace) error {
	// Set user name and email
	if err := git.SetLocalConfig(repoPath, "user.name", ws.Name); err != nil {
		return fmt.Errorf("failed to set user.name: %w", err)
	}

	if err := git.SetLocalConfig(repoPath, "user.email", ws.Email); err != nil {
		return fmt.Errorf("failed to set user.email: %w", err)
	}

	// Set up signing if configured
	switch ws.Signing {
	case "ssh":
		if err := git.SetLocalConfig(repoPath, "gpg.format", "ssh"); err != nil {
			return fmt.Errorf("failed to set gpg.format: %w", err)
		}
		if err := git.SetLocalConfig(repoPath, "user.signingkey", ws.SSHKey+".pub"); err != nil {
			return fmt.Errorf("failed to set signing key: %w", err)
		}
		if err := git.SetLocalConfig(repoPath, "commit.gpgsign", "true"); err != nil {
			return fmt.Errorf("failed to enable commit signing: %w", err)
		}
	case "gpg":
		// Note: GPG key should be set in workspace gitconfig
		if err := git.SetLocalConfig(repoPath, "commit.gpgsign", "true"); err != nil {
			return fmt.Errorf("failed to enable commit signing: %w", err)
		}
	case "none":
		if err := git.SetLocalConfig(repoPath, "commit.gpgsign", "false"); err != nil {
			return fmt.Errorf("failed to disable commit signing: %w", err)
		}
	}

	return nil
}

func getBranchDisplay(branch string) string {
	if branch == "" {
		return "default"
	}
	return branch
}
