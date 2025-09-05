package gws

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gitworkspaces/gitws/internal/git"
	"github.com/gitworkspaces/gitws/internal/prompt"
	"github.com/gitworkspaces/gitws/internal/rewrite"
	"github.com/spf13/cobra"
)

var (
	statusExitNonZero bool
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status [path]",
	Short: "Show repository status and configuration",
	Long: `Show the current repository's status and Git configuration.

This command displays:
- Origin remote URL and resolved alias
- Local user configuration
- Signing status
- Guard hooks status

Examples:
  gitws status
  gitws status /path/to/repo
  gitws status --exit-non-zero`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)

	statusCmd.Flags().BoolVar(&statusExitNonZero, "exit-non-zero", false, "Exit with non-zero code if issues found")
}

func runStatus(cmd *cobra.Command, args []string) error {
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

	// Get remote URL
	remoteURL, err := git.GetRemoteURL(gitRoot)
	if err != nil {
		return fmt.Errorf("failed to get remote URL: %w", err)
	}

	// Get local config
	userName, _ := git.GetLocalConfig(gitRoot, "user.name")
	userEmail, _ := git.GetLocalConfig(gitRoot, "user.email")

	// Get signing status
	signingEnabled, signingMethod, signingKey, _ := git.GetSigningStatus(gitRoot)

	// Check if hooks are installed
	hooksInstalled, _ := git.CheckHooksInstalled(gitRoot)

	// Try to determine workspace from SSH alias
	workspaceName := "unknown"
	realHost := "unknown"
	if strings.HasPrefix(remoteURL, "git@") {
		if host, err := rewrite.ExtractHostFromSSHURL(remoteURL); err == nil {
			realHost = host
			// Try to extract workspace from alias
			if parts := strings.Split(host, "-"); len(parts) > 1 {
				workspaceName = parts[len(parts)-1] // Last part is usually workspace
			}
		}
	}

	// Check for issues
	var issues []string
	if userName == "" {
		issues = append(issues, "No user.name configured")
	}
	if userEmail == "" {
		issues = append(issues, "No user.email configured")
	}
	if !hooksInstalled {
		issues = append(issues, "Guard hooks not installed")
	}

	// Prepare status data
	headers := []string{"Property", "Value"}
	rows := [][]string{
		{"Repository", filepath.Base(gitRoot)},
		{"Path", gitRoot},
		{"Origin", remoteURL},
		{"SSH Alias", realHost},
		{"Workspace", workspaceName},
		{"User Name", getDisplayValue(userName, "Not set")},
		{"User Email", getDisplayValue(userEmail, "Not set")},
		{"Signing", getSigningDisplay(signingEnabled, signingMethod)},
		{"Signing Key", getDisplayValue(signingKey, "Not set")},
		{"Guard Hooks", getBoolDisplay(hooksInstalled)},
	}

	// Show status
	if err := prompt.ShowStatusTable(headers, rows); err != nil {
		return err
	}

	// Show issues if any
	if len(issues) > 0 {
		fmt.Println()
		fmt.Println("⚠️  Issues found:")
		for _, issue := range issues {
			fmt.Printf("   • %s\n", issue)
		}
		fmt.Println()
		fmt.Println("Run 'gitws doctor' for detailed analysis and fixes.")

		if statusExitNonZero {
			os.Exit(1)
		}
	} else {
		fmt.Println()
		fmt.Println("✓ All checks passed!")
	}

	return nil
}

func getDisplayValue(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func getSigningDisplay(enabled bool, method string) string {
	if !enabled {
		return "Disabled"
	}
	return fmt.Sprintf("Enabled (%s)", method)
}

func getBoolDisplay(value bool) string {
	if value {
		return "Installed"
	}
	return "Not installed"
}
