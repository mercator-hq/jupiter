package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Version is the semantic version (set by build flags)
	Version = "0.1.0"
	// GitCommit is the git commit hash (set by build flags)
	GitCommit = "unknown"
	// BuildDate is the build timestamp (set by build flags)
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print detailed version information including Git commit and build date.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Mercator Jupiter %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		fmt.Printf("Build Date: %s\n", BuildDate)
		fmt.Printf("Go Version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
