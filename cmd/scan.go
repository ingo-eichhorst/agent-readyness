package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ingo/agent-readyness/internal/config"
	"github.com/ingo/agent-readyness/internal/pipeline"
	"github.com/ingo/agent-readyness/internal/scoring"
)

var (
	configPath   string
	threshold    float64
	jsonOutput   bool
	noLLM        bool   // Disable LLM features (C4 + C7)
	debug        bool   // Enable debug output (initially C7 only, future: all categories)
	outputHTML   string // Path to output HTML file
	baselinePath string // Path to previous JSON for trend comparison
	badgeOutput  bool   // Generate shields.io badge markdown
	debugDir     string // C7 response persistence directory
)

var scanCmd = &cobra.Command{
	Use:   "scan <directory>",
	Short: "Scan a project for agent readiness",
	Long: `Scan a project directory for agent readiness.

Supported languages: Go, Python, TypeScript
Languages are auto-detected from project files (go.mod, pyproject.toml, tsconfig.json, etc.)
No --lang flag needed.

LLM features (C4 documentation analysis and C7 agent evaluation) are auto-enabled when
Claude CLI is detected. Use --no-llm to disable all LLM features.`,
	Args:  cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("cannot resolve path: %s", err)
		}

		if err := validateProject(dir); err != nil {
			return err
		}

		// --debug-dir implies --debug
		if debugDir != "" {
			debug = true
			absDir, absErr := filepath.Abs(debugDir)
			if absErr != nil {
				return fmt.Errorf("invalid debug-dir path: %w", absErr)
			}
			debugDir = absDir
		}

		// Load scoring config
		cfg, err := scoring.LoadConfig("")
		if err != nil {
			return fmt.Errorf("load scoring config: %w", err)
		}

		// Load project config (.arsrc.yml) and apply overrides
		projectCfg, err := config.LoadProjectConfig(dir, configPath)
		if err != nil {
			return fmt.Errorf("load project config: %w", err)
		}
		if projectCfg != nil {
			projectCfg.ApplyToScoringConfig(cfg)
			// Apply threshold from project config if not set via CLI
			if threshold == 0 && projectCfg.Scoring.Threshold > 0 {
				threshold = projectCfg.Scoring.Threshold
			}
		}

		spinner := pipeline.NewSpinner(os.Stderr)
		onProgress := func(stage, detail string) {
			spinner.Update(detail)
		}
		spinner.Start("Scanning...")

		p := pipeline.New(cmd.OutOrStdout(), verbose, cfg, threshold, jsonOutput, onProgress)

		// Show CLI status and handle LLM feature enablement
		cliStatus := p.GetCLIStatus()
		if cliStatus.Available {
			if noLLM {
				p.DisableLLM()
				fmt.Fprintf(cmd.OutOrStdout(), "LLM features disabled (--no-llm flag)\n")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Claude CLI detected (%s) - LLM features enabled\n", cliStatus.Version)
			}
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Claude CLI not found - LLM features disabled\n")
			if verbose {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", cliStatus.InstallHint)
			}
		}

		// Configure debug output
		if debug {
			p.SetC7Debug(true)
		}

		// Configure C7 response persistence/replay
		if debugDir != "" {
			// Silently ignore if --no-llm is set
			if !noLLM {
				p.SetDebugDir(debugDir)
			}
		}

		// Configure HTML output if requested
		if outputHTML != "" {
			p.SetHTMLOutput(outputHTML, baselinePath)
		}

		// Configure badge output if requested
		if badgeOutput {
			p.SetBadgeOutput(true)
		}

		err = p.Run(dir)
		if err != nil {
			spinner.Stop("") // clear spinner before error
			return err
		}
		spinner.Stop("Done.")

		// Show HTML output path if generated
		if outputHTML != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "\nHTML report generated: %s\n", outputHTML)
		}

		return nil
	},
}

func init() {
	scanCmd.Flags().StringVar(&configPath, "config", "", "path to .arsrc.yml project config file")
	scanCmd.Flags().Float64Var(&threshold, "threshold", 0, "minimum composite score (exit code 2 if below)")
	scanCmd.Flags().BoolVar(&jsonOutput, "json", false, "output results as JSON")
	scanCmd.Flags().BoolVar(&noLLM, "no-llm", false, "disable LLM features (C4 documentation analysis and C7 agent evaluation)")
	scanCmd.Flags().BoolVar(&debug, "debug", false, "enable verbose debug output")
	scanCmd.Flags().StringVar(&outputHTML, "output-html", "", "generate self-contained HTML report at specified path")
	scanCmd.Flags().StringVar(&baselinePath, "baseline", "", "path to previous JSON output for trend comparison")
	scanCmd.Flags().BoolVar(&badgeOutput, "badge", false, "generate shields.io badge markdown URL")
	scanCmd.Flags().StringVar(&debugDir, "debug-dir", "", "directory for C7 response persistence and replay")
	rootCmd.AddCommand(scanCmd)
}

// validateProject checks that dir exists, is a directory, and contains recognized source files.
func validateProject(dir string) error {
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

	// Check for any recognized project indicator
	indicators := []string{
		"go.mod",            // Go
		"pyproject.toml",    // Python
		"setup.py",          // Python
		"requirements.txt",  // Python
		"tsconfig.json",     // TypeScript
		"package.json",      // JavaScript/TypeScript
	}

	for _, f := range indicators {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			return nil
		}
	}

	// Fallback: check for any recognized source file
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("cannot read directory: %s", err)
	}
	recognizedExts := map[string]bool{".go": true, ".py": true, ".ts": true, ".tsx": true}
	for _, entry := range entries {
		if !entry.IsDir() {
			ext := filepath.Ext(entry.Name())
			if recognizedExts[ext] {
				return nil
			}
		}
	}

	return fmt.Errorf("no recognized project found in: %s\nSupported: Go (go.mod), Python (pyproject.toml), TypeScript (tsconfig.json)\nEnsure the directory contains source files (.go, .py, .ts)", dir)
}
