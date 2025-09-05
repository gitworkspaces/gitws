package cli

import (
	"fmt"
	"os"

	"github.com/gitworkspaces/gitws/internal/config"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	verbose    bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gitws",
	Short: "Git workspace manager - never mix work/personal git again",
	Long: `gitws helps you manage separate Git identities for different workspaces.
It creates per-workspace SSH keys, configures SSH aliases, and ensures
proper Git configuration isolation.

Examples:
  gitws init work --email you@work.com --host github
  gitws init personal --email you@me.com --host github
  gitws clone work microsoft/vscode
  gitws status
  gitws doctor`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Ensure config directory exists
		configDir, err := config.ConfigDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to create config directory: %v\n", err)
			os.Exit(1)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute(version string) error {
	rootCmd.Version = version
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
}
