package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ingo/agent-readyness/internal/pipeline"
)

var scanCmd = &cobra.Command{
	Use:   "scan <directory>",
	Short: "Scan a Go project for agent readiness",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("cannot resolve path: %s", err)
		}

		if err := validateGoProject(dir); err != nil {
			return err
		}

		p := pipeline.New(cmd.OutOrStdout(), verbose)
		return p.Run(dir)
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
}

// validateGoProject checks that dir exists, is a directory, and contains a Go project.
func validateGoProject(dir string) error {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory not found: %s", dir)
	}
	if err != nil {
		return fmt.Errorf("cannot access directory: %s", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", dir)
	}

	// Check for go.mod
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return nil
	}

	// Fallback: check for any .go file
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("cannot read directory: %s", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".go" {
			return nil
		}
	}

	return fmt.Errorf("not a Go project: %s\nNo go.mod file or .go source files found. Please specify a directory containing a Go project.", dir)
}
