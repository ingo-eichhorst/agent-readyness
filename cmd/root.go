package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
	"github.com/ingo-eichhorst/agent-readyness/pkg/version"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:     "ars",
	Short:   "Agent Readiness Score - analyze Go codebases for AI agent compatibility",
	Long:    "ARS analyzes Go codebases and produces a composite score measuring how well\nthe repository supports AI agent workflows. It evaluates code health,\narchitectural navigability, and testing infrastructure.",
	Version: version.Version,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.SilenceErrors = true
}

// Execute runs the root command and exits with code 1 on error.
// ExitError is handled specially: its Code is used as the exit code.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		var exitErr *types.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		os.Exit(1)
	}
}
