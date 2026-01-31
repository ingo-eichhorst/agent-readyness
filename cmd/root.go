package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:     "ars",
	Short:   "Agent Readiness Score - analyze Go codebases for AI agent compatibility",
	Long:    "ARS analyzes Go codebases and produces a composite score measuring how well\nthe repository supports AI agent workflows. It evaluates code health,\narchitectural navigability, and testing infrastructure.",
	Version: version,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
}

// Execute runs the root command and exits with code 1 on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
