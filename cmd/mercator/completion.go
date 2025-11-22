package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for Mercator.

To load completions:

Bash:
  $ source <(mercator completion bash)
  # To load permanently:
  $ mercator completion bash > /etc/bash_completion.d/mercator

Zsh:
  $ mercator completion zsh > "${fpath[1]}/_mercator"
  $ compinit

Fish:
  $ mercator completion fish | source
  # To load permanently:
  $ mercator completion fish > ~/.config/fish/completions/mercator.fish

PowerShell:
  PS> mercator completion powershell | Out-String | Invoke-Expression
  # To load permanently, add to your PowerShell profile
`,
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Args:      cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletion(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
