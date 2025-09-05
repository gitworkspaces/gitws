package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gitworkspaces/gitws/internal/config"
	"github.com/gitworkspaces/gitws/internal/fsutil"
	"github.com/gitworkspaces/gitws/internal/prompt"
	"github.com/gitworkspaces/gitws/internal/ssh"
	"github.com/gitworkspaces/gitws/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	initEmail     string
	initHost      string
	initHostName  string
	initRoot      string
	initSigning   string
	initName      string
	initForce     bool
	initRotateKey bool
	initGPGKey    string
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init <workspace>",
	Short: "Initialize a new Git workspace",
	Long: `Initialize a new Git workspace with separate SSH keys and configuration.

This command will:
- Generate a new SSH key pair for the workspace
- Configure SSH aliases in ~/.ssh/config
- Set up Git configuration isolation
- Create workspace-specific settings

Examples:
  gitws init work --email you@work.com --host github
  gitws init personal --email you@me.com --host github --signing ssh
  gitws init client --email you@client.com --host-name gitlab.client.com`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVar(&initEmail, "email", "", "Email address for this workspace (required)")
	initCmd.Flags().StringVar(&initHost, "host", "", "Git provider (github, gitlab, bitbucket)")
	initCmd.Flags().StringVar(&initHostName, "host-name", "", "Custom hostname (mutually exclusive with --host)")
	initCmd.Flags().StringVar(&initRoot, "root", "", "Workspace root directory (default: ~/code/<workspace>)")
	initCmd.Flags().StringVar(&initSigning, "signing", "none", "Signing method (none, ssh, gpg)")
	initCmd.Flags().StringVar(&initName, "name", "", "Display name (defaults to workspace name or $USER)")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing managed blocks")
	initCmd.Flags().BoolVar(&initRotateKey, "rotate-key", false, "Generate new SSH key even if one exists")
	initCmd.Flags().StringVar(&initGPGKey, "gpg-key", "", "GPG key ID for signing (required with --signing gpg)")

	initCmd.MarkFlagRequired("email")
	initCmd.MarkFlagsMutuallyExclusive("host", "host-name")
}

func runInit(cmd *cobra.Command, args []string) error {
	workspaceName := args[0]

	// Validate inputs
	if initHost == "" && initHostName == "" {
		return fmt.Errorf("either --host or --host-name must be specified")
	}

	if initSigning == "gpg" && initGPGKey == "" {
		return fmt.Errorf("--gpg-key is required when using --signing gpg")
	}

	// Resolve hostname
	var hostName string
	if initHost != "" {
		if host, exists := workspace.ProviderHosts[initHost]; exists {
			hostName = host
		} else {
			return fmt.Errorf("unknown provider: %s (supported: github, gitlab, bitbucket)", initHost)
		}
	} else {
		hostName = initHostName
	}

	// Build SSH alias
	providerOrHost := initHost
	if providerOrHost == "" {
		providerOrHost = initHostName
	}
	alias := workspace.BuildSSHAlias(providerOrHost, workspaceName)

	// Set default root if not provided
	root := initRoot
	if root == "" {
		var err error
		root, err = workspace.DefaultRoot(workspaceName)
		if err != nil {
			return fmt.Errorf("failed to get default root: %w", err)
		}
	}

	// Expand root path
	expandedRoot, err := workspace.ExpandPath(root)
	if err != nil {
		return fmt.Errorf("failed to expand root path: %w", err)
	}

	// Set display name
	displayName := initName
	if displayName == "" {
		displayName = workspaceName
		if user := os.Getenv("USER"); user != "" {
			displayName = user
		}
	}

	// Load existing config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if workspace already exists
	if _, exists := cfg.GetWorkspace(workspaceName); exists && !initForce {
		return fmt.Errorf("workspace %q already exists (use --force to overwrite)", workspaceName)
	}

	// Generate SSH key
	privPath, pubPath, keyCreated, err := ssh.EnsureKey(workspaceName, initEmail)
	if err != nil {
		return fmt.Errorf("failed to ensure SSH key: %w", err)
	}

	// Rotate key if requested
	if initRotateKey && !keyCreated {
		// TODO: Implement key rotation with backup
		return fmt.Errorf("key rotation not yet implemented")
	}

	// Update SSH config
	if err := ssh.UpsertSSHConfigBlock(workspaceName, alias, hostName, privPath); err != nil {
		return fmt.Errorf("failed to update SSH config: %w", err)
	}

	// Update global gitconfig with includeIf
	if err := updateGlobalGitConfig(workspaceName, expandedRoot); err != nil {
		return fmt.Errorf("failed to update global gitconfig: %w", err)
	}

	// Create workspace gitconfig
	if err := createWorkspaceGitConfig(workspaceName, displayName, initEmail, initSigning, privPath, initGPGKey); err != nil {
		return fmt.Errorf("failed to create workspace gitconfig: %w", err)
	}

	// Save workspace config
	ws := config.Workspace{
		Email:    initEmail,
		Provider: initHost,
		HostName: hostName,
		SSHAlias: alias,
		SSHKey:   privPath,
		Root:     expandedRoot,
		Signing:  initSigning,
		Name:     displayName,
	}
	cfg.SetWorkspace(workspaceName, ws)

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Get public key for display
	publicKey, err := ssh.GetPublicKey(pubPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	// Show summary
	summary := prompt.SummaryData{
		Title: fmt.Sprintf("✓ Workspace '%s' initialized successfully", workspaceName),
		Items: []prompt.SummaryItem{
			{Label: "SSH Alias", Value: alias, Icon: "🔑"},
			{Label: "Host", Value: hostName, Icon: "🌐"},
			{Label: "Root", Value: expandedRoot, Icon: "📁"},
			{Label: "Email", Value: initEmail, Icon: "📧"},
			{Label: "Signing", Value: initSigning, Icon: "✍️"},
		},
		PublicKey: publicKey,
		NextSteps: []string{
			fmt.Sprintf("Add the public key to your %s account", hostName),
			fmt.Sprintf("Use 'gitws clone %s ORG/REPO' to clone repositories", workspaceName),
			"Run 'gitws status' to check repository configuration",
		},
	}

	return prompt.ShowSummary(summary)
}

func updateGlobalGitConfig(workspaceName, root string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	gitConfigPath := filepath.Join(home, ".gitconfig")

	// Read existing config
	var content string
	if fsutil.FileExists(gitConfigPath) {
		data, err := os.ReadFile(gitConfigPath)
		if err != nil {
			return fmt.Errorf("failed to read gitconfig: %w", err)
		}
		content = string(data)
	}

	// Create backup
	if err := fsutil.CreateBackup(gitConfigPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Build includeIf condition
	condition, err := workspace.BuildIncludeIfCondition(root)
	if err != nil {
		return fmt.Errorf("failed to build includeIf condition: %w", err)
	}

	// Get gitconfig path
	gitConfigWorkspacePath, err := workspace.GitConfigPath(workspaceName)
	if err != nil {
		return fmt.Errorf("failed to get workspace gitconfig path: %w", err)
	}

	// Build new block
	startMarker := workspace.IncludeIfStartMarker()
	endMarker := workspace.IncludeIfEndMarker()

	newBlock := fmt.Sprintf(`%s
[includeIf "%s"]
  path = %s
%s`, startMarker, condition, gitConfigWorkspacePath, endMarker)

	// Replace content between markers
	newContent, _ := fsutil.ReplaceBetweenMarkers(content, startMarker, endMarker, newBlock)

	// Write updated config
	if err := fsutil.AtomicWrite(gitConfigPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write gitconfig: %w", err)
	}

	return nil
}

func createWorkspaceGitConfig(workspaceName, displayName, email, signing, keyPath, gpgKey string) error {
	// Ensure directory exists
	gitConfigPath, err := workspace.GitConfigPath(workspaceName)
	if err != nil {
		return fmt.Errorf("failed to get gitconfig path: %w", err)
	}

	dir := filepath.Dir(gitConfigPath)
	if err := fsutil.EnsureDir(dir); err != nil {
		return fmt.Errorf("failed to create gitconfig directory: %w", err)
	}

	// Build gitconfig content
	var content strings.Builder

	content.WriteString("[user]\n")
	content.WriteString(fmt.Sprintf("  name = %s\n", displayName))
	content.WriteString(fmt.Sprintf("  email = %s\n", email))
	content.WriteString("\n")

	content.WriteString("[commit]\n")
	content.WriteString("  gpgsign = false\n")
	content.WriteString("\n")

	// Add signing configuration
	switch signing {
	case "ssh":
		content.WriteString("[gpg]\n")
		content.WriteString("  format = ssh\n")
		content.WriteString("\n")
		content.WriteString("[user]\n")
		content.WriteString(fmt.Sprintf("  signingkey = %s.pub\n", keyPath))
		content.WriteString("\n")
		content.WriteString("[commit]\n")
		content.WriteString("  gpgsign = true\n")
		content.WriteString("\n")
	case "gpg":
		content.WriteString("[user]\n")
		content.WriteString(fmt.Sprintf("  signingkey = %s\n", gpgKey))
		content.WriteString("\n")
		content.WriteString("[commit]\n")
		content.WriteString("  gpgsign = true\n")
		content.WriteString("\n")
	}

	// Write gitconfig
	if err := fsutil.AtomicWrite(gitConfigPath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write workspace gitconfig: %w", err)
	}

	return nil
}
