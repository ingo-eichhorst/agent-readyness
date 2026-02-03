package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ingo/agent-readyness/internal/config"
	"github.com/ingo/agent-readyness/internal/llm"
	"github.com/ingo/agent-readyness/internal/pipeline"
	"github.com/ingo/agent-readyness/internal/scoring"
)

var (
	configPath  string
	threshold   float64
	jsonOutput  bool
	enableC4LLM bool
)

var scanCmd = &cobra.Command{
	Use:   "scan <directory>",
	Short: "Scan a project for agent readiness",
	Long: `Scan a project directory for agent readiness.

Supported languages: Go, Python, TypeScript
Languages are auto-detected from project files (go.mod, pyproject.toml, tsconfig.json, etc.)
No --lang flag needed.`,
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

		// Handle LLM-based C4 analysis if enabled
		var llmClient *llm.Client
		if enableC4LLM {
			apiKey := os.Getenv("ANTHROPIC_API_KEY")
			if apiKey == "" {
				return fmt.Errorf("--enable-c4-llm requires ANTHROPIC_API_KEY environment variable\nGet your API key from: https://console.anthropic.com/")
			}

			// Show cost estimate and get user confirmation
			// Estimate based on typical project size
			estimate := llm.EstimateCost(500, 5) // ~500 word README, ~5 files sampled
			fmt.Fprintf(cmd.OutOrStdout(), "\nLLM Analysis Cost Estimate\n")
			fmt.Fprintf(cmd.OutOrStdout(), "==========================\n")
			fmt.Fprintf(cmd.OutOrStdout(), "Files to analyze: ~%d (README + examples)\n", estimate.FilesCount)
			fmt.Fprintf(cmd.OutOrStdout(), "Estimated cost: %s\n\n", estimate.FormatCost())
			fmt.Fprintf(cmd.OutOrStdout(), "This will send documentation content to Anthropic's API.\n")
			fmt.Fprintf(cmd.OutOrStdout(), "Continue? (yes/no): ")

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response != "yes" && response != "y" {
				fmt.Fprintf(cmd.OutOrStdout(), "LLM analysis cancelled. Running static analysis only.\n\n")
				enableC4LLM = false
			} else {
				var err error
				llmClient, err = llm.NewClient(apiKey)
				if err != nil {
					return fmt.Errorf("failed to create LLM client: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "\n")
			}
		}

		spinner := pipeline.NewSpinner(os.Stderr)
		onProgress := func(stage, detail string) {
			spinner.Update(detail)
		}
		spinner.Start("Scanning...")

		p := pipeline.New(cmd.OutOrStdout(), verbose, cfg, threshold, jsonOutput, onProgress)
		if llmClient != nil {
			p.SetLLMClient(llmClient)
		}
		err = p.Run(dir)
		if err != nil {
			spinner.Stop("") // clear spinner before error
			return err
		}
		spinner.Stop("Done.")
		return nil
	},
}

func init() {
	scanCmd.Flags().StringVar(&configPath, "config", "", "path to .arsrc.yml project config file")
	scanCmd.Flags().Float64Var(&threshold, "threshold", 0, "minimum composite score (exit code 2 if below)")
	scanCmd.Flags().BoolVar(&jsonOutput, "json", false, "output results as JSON")
	scanCmd.Flags().BoolVar(&enableC4LLM, "enable-c4-llm", false, "enable LLM-based C4 content quality evaluation (requires ANTHROPIC_API_KEY)")
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
