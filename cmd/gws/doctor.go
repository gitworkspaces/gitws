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

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor [path]",
	Short: "Diagnose repository configuration issues",
	Long: `Diagnose and report configuration issues in the current repository.

This command checks for:
- Identity mismatches
- Remote URL issues
- Signing configuration problems
- Missing guard hooks
- Workspace configuration issues

Examples:
  gitws doctor
  gitws doctor /path/to/repo`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
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

	// Run all checks
	issues := runAllChecks(gitRoot)

	// Show doctor report
	if err := prompt.ShowDoctorReport(issues); err != nil {
		return err
	}

	// Exit with non-zero if issues found
	if len(issues) > 0 {
		os.Exit(1)
	}

	return nil
}

func runAllChecks(gitRoot string) []prompt.Issue {
	var issues []prompt.Issue

	// Check 1: Git repository validity
	issues = append(issues, checkGitRepository(gitRoot)...)

	// Check 2: Remote configuration
	issues = append(issues, checkRemoteConfiguration(gitRoot)...)

	// Check 3: User identity
	issues = append(issues, checkUserIdentity(gitRoot)...)

	// Check 4: Signing configuration
	issues = append(issues, checkSigningConfiguration(gitRoot)...)

	// Check 5: Guard hooks
	issues = append(issues, checkGuardHooks(gitRoot)...)

	// Check 6: Workspace consistency
	issues = append(issues, checkWorkspaceConsistency(gitRoot)...)

	return issues
}

func checkGitRepository(gitRoot string) []prompt.Issue {
	var issues []prompt.Issue

	// Check git version
	version, err := git.CheckGitPresence()
	if err != nil {
		issues = append(issues, prompt.Issue{
			Type:    "error",
			Message: "Git is not installed or not in PATH",
			Fix:     "Install Git and ensure it's in your PATH",
		})
	} else if verbose {
		// Add info about git version
		issues = append(issues, prompt.Issue{
			Type:    "info",
			Message: fmt.Sprintf("Git version: %s", version),
			Fix:     "",
		})
	}

	return issues
}

func checkRemoteConfiguration(gitRoot string) []prompt.Issue {
	var issues []prompt.Issue

	remoteURL, err := git.GetRemoteURL(gitRoot)
	if err != nil {
		issues = append(issues, prompt.Issue{
			Type:    "error",
			Message: "No origin remote configured",
			Fix:     "Add origin remote: git remote add origin <url>",
		})
		return issues
	}

	// Check if using SSH
	if !strings.HasPrefix(remoteURL, "git@") {
		issues = append(issues, prompt.Issue{
			Type:    "warning",
			Message: "Remote URL is not using SSH",
			Fix:     "Use 'gitws fix' to rewrite remote URL to SSH",
		})
	}

	// Check if using gitws alias
	if strings.HasPrefix(remoteURL, "git@") {
		host, err := rewrite.ExtractHostFromSSHURL(remoteURL)
		if err == nil {
			if !strings.Contains(host, "gws") && !strings.Contains(host, "gitws") {
				issues = append(issues, prompt.Issue{
					Type:    "warning",
					Message: fmt.Sprintf("Remote URL not using gitws alias (current: %s)", host),
					Fix:     "Use 'gitws fix' to rewrite remote URL to use workspace alias",
				})
			}
		}
	}

	return issues
}

func checkUserIdentity(gitRoot string) []prompt.Issue {
	var issues []prompt.Issue

	userName, err := git.GetLocalConfig(gitRoot, "user.name")
	if err != nil || userName == "" {
		issues = append(issues, prompt.Issue{
			Type:    "error",
			Message: "No user.name configured",
			Fix:     "Set user.name: git config user.name 'Your Name'",
		})
	}

	userEmail, err := git.GetLocalConfig(gitRoot, "user.email")
	if err != nil || userEmail == "" {
		issues = append(issues, prompt.Issue{
			Type:    "error",
			Message: "No user.email configured",
			Fix:     "Set user.email: git config user.email 'your@email.com'",
		})
	}

	return issues
}

func checkSigningConfiguration(gitRoot string) []prompt.Issue {
	var issues []prompt.Issue

	signingEnabled, signingMethod, signingKey, err := git.GetSigningStatus(gitRoot)
	if err != nil {
		issues = append(issues, prompt.Issue{
			Type:    "warning",
			Message: "Could not determine signing configuration",
			Fix:     "Check your Git signing configuration",
		})
		return issues
	}

	if signingEnabled {
		if signingKey == "" {
			issues = append(issues, prompt.Issue{
				Type:    "error",
				Message: "Signing enabled but no signing key configured",
				Fix:     "Configure signing key: git config user.signingkey <key>",
			})
		}

		if signingMethod == "ssh" {
			// Check if SSH key exists
			if signingKey != "" && !strings.HasSuffix(signingKey, ".pub") {
				issues = append(issues, prompt.Issue{
					Type:    "warning",
					Message: "SSH signing key should end with .pub",
					Fix:     "Update signing key to use .pub file",
				})
			}
		}
	}

	return issues
}

func checkGuardHooks(gitRoot string) []prompt.Issue {
	var issues []prompt.Issue

	hooksInstalled, err := git.CheckHooksInstalled(gitRoot)
	if err != nil {
		issues = append(issues, prompt.Issue{
			Type:    "warning",
			Message: "Could not check guard hooks status",
			Fix:     "Manually verify hooks in .git/hooks/",
		})
		return issues
	}

	if !hooksInstalled {
		issues = append(issues, prompt.Issue{
			Type:    "warning",
			Message: "Guard hooks not installed",
			Fix:     "Use 'gitws fix --enable-guards' to install hooks",
		})
	}

	return issues
}

func checkWorkspaceConsistency(gitRoot string) []prompt.Issue {
	var issues []prompt.Issue

	// Try to determine workspace from remote URL
	remoteURL, err := git.GetRemoteURL(gitRoot)
	if err != nil {
		return issues // Already handled in remote check
	}

	if !strings.HasPrefix(remoteURL, "git@") {
		return issues // Not SSH, skip workspace check
	}

	host, err := rewrite.ExtractHostFromSSHURL(remoteURL)
	if err != nil {
		return issues
	}

	// Try to find workspace in config
	cfg, err := config.Load()
	if err != nil {
		issues = append(issues, prompt.Issue{
			Type:    "warning",
			Message: "Could not load workspace configuration",
			Fix:     "Check ~/.gws/config.yaml",
		})
		return issues
	}

	// Find workspace by SSH alias
	var foundWorkspace string
	for name, ws := range cfg.Workspaces {
		if ws.SSHAlias == host {
			foundWorkspace = name
			break
		}
	}

	if foundWorkspace == "" {
		issues = append(issues, prompt.Issue{
			Type:    "warning",
			Message: fmt.Sprintf("SSH alias '%s' not found in workspace configuration", host),
			Fix:     "Run 'gitws init' to create workspace or check configuration",
		})
		return issues
	}

	// Check if repository is in expected workspace root
	ws := cfg.Workspaces[foundWorkspace]
	if !strings.HasPrefix(gitRoot, ws.Root) {
		issues = append(issues, prompt.Issue{
			Type:    "warning",
			Message: fmt.Sprintf("Repository not in workspace root (expected: %s)", ws.Root),
			Fix:     "Move repository to workspace root or update workspace configuration",
		})
	}

	return issues
}
