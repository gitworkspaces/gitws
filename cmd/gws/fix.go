package gws

import (
	"fmt"
	"os"
	"strings"

	"github.com/gitworkspaces/gitws/internal/config"
	"github.com/gitworkspaces/gitws/internal/git"
	"github.com/gitworkspaces/gitws/internal/prompt"
	"github.com/gitworkspaces/gitws/internal/rewrite"
	"github.com/spf13/cobra"
)

var (
	fixYes           bool
	fixEnableGuards  bool
	fixRewriteRemote bool
	fixSetIdentity   bool
)

// fixCmd represents the fix command
var fixCmd = &cobra.Command{
	Use:   "fix [path]",
	Short: "Fix repository configuration issues",
	Long: `Fix common repository configuration issues.

This command can:
- Rewrite remote URL to use workspace SSH alias
- Set proper user identity configuration
- Install guard hooks to prevent identity mixing

Examples:
  gitws fix
  gitws fix /path/to/repo --yes --enable-guards
  gitws fix --rewrite-remote --set-identity`,
	Args: cobra.MaximumNArgs(1),
	RunE: runFix,
}

func init() {
	rootCmd.AddCommand(fixCmd)

	fixCmd.Flags().BoolVar(&fixYes, "yes", false, "Skip confirmation prompts")
	fixCmd.Flags().BoolVar(&fixEnableGuards, "enable-guards", false, "Install guard hooks")
	fixCmd.Flags().BoolVar(&fixRewriteRemote, "rewrite-remote", false, "Rewrite remote URL to use workspace alias")
	fixCmd.Flags().BoolVar(&fixSetIdentity, "set-identity", false, "Set user identity from workspace config")
}

func runFix(cmd *cobra.Command, args []string) error {
	var repoPath string
	var err error

	if len(args) > 0 {
		repoPath = args[0]
	} else {
		repoPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Find git root
	gitRoot, err := git.FindGitRoot(repoPath)
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Load workspace config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Determine what to fix
	var fixes []string
	var changes []string

	// Check remote URL
	remoteURL, err := git.GetRemoteURL(gitRoot)
	if err == nil {
		workspace, needsRewrite := checkRemoteURL(remoteURL, cfg)
		if needsRewrite && (fixRewriteRemote || !fixYes) {
			fixes = append(fixes, "rewrite-remote")
			if workspace != "" {
				changes = append(changes, fmt.Sprintf("Rewrite remote URL to use workspace '%s' alias", workspace))
			}
		}
	}

	// Check user identity
	userName, _ := git.GetLocalConfig(gitRoot, "user.name")
	userEmail, _ := git.GetLocalConfig(gitRoot, "user.email")
	if (userName == "" || userEmail == "") && (fixSetIdentity || !fixYes) {
		fixes = append(fixes, "set-identity")
		changes = append(changes, "Set user identity from workspace configuration")
	}

	// Check guard hooks
	hooksInstalled, _ := git.CheckHooksInstalled(gitRoot)
	if !hooksInstalled && (fixEnableGuards || !fixYes) {
		fixes = append(fixes, "enable-guards")
		changes = append(changes, "Install guard hooks")
	}

	if len(fixes) == 0 {
		fmt.Println("✓ No fixes needed. Repository is properly configured.")
		return nil
	}

	// Show what will be fixed
	fmt.Println("The following changes will be made:")
	for i, change := range changes {
		fmt.Printf("%d. %s\n", i+1, change)
	}
	fmt.Println()

	// Confirm unless --yes
	if !fixYes {
		confirmed, err := prompt.Confirm("Apply these fixes?")
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirmed {
			fmt.Println("Fix cancelled.")
			return nil
		}
	}

	// Apply fixes
	var appliedFixes []string

	for _, fix := range fixes {
		switch fix {
		case "rewrite-remote":
			if err := applyRewriteRemote(gitRoot, cfg); err != nil {
				fmt.Printf("❌ Failed to rewrite remote: %v\n", err)
			} else {
				appliedFixes = append(appliedFixes, "Remote URL rewritten")
			}

		case "set-identity":
			if err := applySetIdentity(gitRoot, cfg); err != nil {
				fmt.Printf("❌ Failed to set identity: %v\n", err)
			} else {
				appliedFixes = append(appliedFixes, "User identity set")
			}

		case "enable-guards":
			if err := applyEnableGuards(gitRoot); err != nil {
				fmt.Printf("❌ Failed to install guard hooks: %v\n", err)
			} else {
				appliedFixes = append(appliedFixes, "Guard hooks installed")
			}
		}
	}

	// Show summary
	if len(appliedFixes) > 0 {
		fmt.Println()
		fmt.Println("✓ Applied fixes:")
		for _, fix := range appliedFixes {
			fmt.Printf("   • %s\n", fix)
		}
		fmt.Println()
		fmt.Println("Run 'gitws status' to verify the changes.")
	}

	return nil
}

func checkRemoteURL(remoteURL string, cfg *config.File) (string, bool) {
	if !strings.HasPrefix(remoteURL, "git@") {
		return "", true // Needs rewrite to SSH
	}

	host, err := rewrite.ExtractHostFromSSHURL(remoteURL)
	if err != nil {
		return "", true // Needs rewrite
	}

	// Check if this is already a gitws alias
	if strings.Contains(host, "gws") || strings.Contains(host, "gitws") {
		return "", false // Already using gitws alias
	}

	// Try to find matching workspace
	for name, ws := range cfg.Workspaces {
		if strings.Contains(host, ws.HostName) {
			return name, true // Found workspace, needs rewrite
		}
	}

	return "", false // No workspace found, leave as is
}

func applyRewriteRemote(gitRoot string, cfg *config.File) error {
	remoteURL, err := git.GetRemoteURL(gitRoot)
	if err != nil {
		return fmt.Errorf("failed to get remote URL: %w", err)
	}

	// Parse the URL to get org/repo
	org, repo, _, err := rewrite.RewriteURL(remoteURL, "dummy")
	if err != nil {
		return fmt.Errorf("failed to parse remote URL: %w", err)
	}

	// Find the appropriate workspace
	var targetWorkspace config.Workspace
	var found bool

	// Try to match by hostname
	if strings.HasPrefix(remoteURL, "git@") {
		host, err := rewrite.ExtractHostFromSSHURL(remoteURL)
		if err == nil {
			for _, ws := range cfg.Workspaces {
				if host == ws.HostName {
					targetWorkspace = ws
					found = true
					break
				}
			}
		}
	}

	// If not found by hostname, try to match by provider
	if !found {
		for _, ws := range cfg.Workspaces {
			if ws.Provider != "" {
				// This is a provider-based workspace
				targetWorkspace = ws
				found = true
				break
			}
		}
	}

	if !found {
		return fmt.Errorf("no suitable workspace found for remote URL")
	}

	// Build new SSH URL
	newURL := fmt.Sprintf("git@%s:%s/%s.git", targetWorkspace.SSHAlias, org, repo)

	// Update remote
	if err := git.SetRemoteURL(gitRoot, newURL); err != nil {
		return fmt.Errorf("failed to set remote URL: %w", err)
	}

	fmt.Printf("✓ Rewritten remote URL: %s\n", newURL)
	return nil
}

func applySetIdentity(gitRoot string, cfg *config.File) error {
	// Find workspace by repository path
	var targetWorkspace config.Workspace
	var found bool

	for _, ws := range cfg.Workspaces {
		if strings.HasPrefix(gitRoot, ws.Root) {
			targetWorkspace = ws
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("no workspace found for repository path")
	}

	// Set user identity
	if err := git.SetLocalConfig(gitRoot, "user.name", targetWorkspace.Name); err != nil {
		return fmt.Errorf("failed to set user.name: %w", err)
	}

	if err := git.SetLocalConfig(gitRoot, "user.email", targetWorkspace.Email); err != nil {
		return fmt.Errorf("failed to set user.email: %w", err)
	}

	// Set up signing if configured
	switch targetWorkspace.Signing {
	case "ssh":
		if err := git.SetLocalConfig(gitRoot, "gpg.format", "ssh"); err != nil {
			return fmt.Errorf("failed to set gpg.format: %w", err)
		}
		if err := git.SetLocalConfig(gitRoot, "user.signingkey", targetWorkspace.SSHKey+".pub"); err != nil {
			return fmt.Errorf("failed to set signing key: %w", err)
		}
		if err := git.SetLocalConfig(gitRoot, "commit.gpgsign", "true"); err != nil {
			return fmt.Errorf("failed to enable commit signing: %w", err)
		}
	case "gpg":
		if err := git.SetLocalConfig(gitRoot, "commit.gpgsign", "true"); err != nil {
			return fmt.Errorf("failed to enable commit signing: %w", err)
		}
	case "none":
		if err := git.SetLocalConfig(gitRoot, "commit.gpgsign", "false"); err != nil {
			return fmt.Errorf("failed to disable commit signing: %w", err)
		}
	}

	fmt.Printf("✓ Set user identity: %s <%s>\n", targetWorkspace.Name, targetWorkspace.Email)
	return nil
}

func applyEnableGuards(gitRoot string) error {
	if err := git.InstallHooks(gitRoot); err != nil {
		return fmt.Errorf("failed to install hooks: %w", err)
	}

	fmt.Println("✓ Installed guard hooks")
	return nil
}
