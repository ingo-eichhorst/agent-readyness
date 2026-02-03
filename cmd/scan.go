package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ingo/agent-readyness/internal/agent"
	"github.com/ingo/agent-readyness/internal/config"
	"github.com/ingo/agent-readyness/internal/llm"
	"github.com/ingo/agent-readyness/internal/pipeline"
	"github.com/ingo/agent-readyness/internal/scoring"
)

var (
	configPath   string
	threshold    float64
	jsonOutput   bool
	noLLM        bool   // Disable LLM features even when CLI available
	enableC7     bool   // Enable C7 agent evaluation
	outputHTML   string // Path to output HTML file
	baselinePath string // Path to previous JSON for trend comparison
	badgeOutput  bool   // Generate shields.io badge markdown
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

		// Handle C7 agent evaluation if enabled
		if enableC7 {
			// Check Claude CLI availability first
			if err := agent.CheckClaudeCLI(); err != nil {
				return fmt.Errorf("--enable-c7 requires Claude Code CLI to be installed\n%s", err)
			}

			// LLM client needed for scoring (uses ANTHROPIC_API_KEY)
			apiKey := os.Getenv("ANTHROPIC_API_KEY")
			if apiKey == "" {
				return fmt.Errorf("--enable-c7 requires ANTHROPIC_API_KEY environment variable for scoring\nGet your API key from: https://console.anthropic.com/")
			}

			// Show cost estimate and get confirmation
			estimate := llm.EstimateC7Cost()
			fmt.Fprintf(cmd.OutOrStdout(), "\nC7 Agent Evaluation Cost Estimate\n")
			fmt.Fprintf(cmd.OutOrStdout(), "==================================\n")
			fmt.Fprintf(cmd.OutOrStdout(), "Tasks to run: 4 (intent clarity, modification confidence, cross-file coherence, semantic completeness)\n")
			fmt.Fprintf(cmd.OutOrStdout(), "Estimated cost: %s\n", estimate.FormatCost())
			fmt.Fprintf(cmd.OutOrStdout(), "Estimated duration: 5-20 minutes (depends on codebase size)\n\n")
			fmt.Fprintf(cmd.OutOrStdout(), "This will:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  1. Run Claude Code headless against your codebase\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  2. Send agent responses to Anthropic API for scoring\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  3. Use your ANTHROPIC_API_KEY for both operations\n\n")
			fmt.Fprintf(cmd.OutOrStdout(), "Continue? (yes/no): ")

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response != "yes" && response != "y" {
				fmt.Fprintf(cmd.OutOrStdout(), "C7 evaluation cancelled. Running other analyzers only.\n\n")
				enableC7 = false
			} else {
				// Create LLM client for scoring if not already created
				if llmClient == nil {
					var err error
					llmClient, err = llm.NewClient(apiKey)
					if err != nil {
						return fmt.Errorf("failed to create LLM client: %w", err)
					}
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

		// Enable C7 if requested and user confirmed
		if enableC7 && llmClient != nil {
			p.SetC7Enabled(llmClient)
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
	scanCmd.Flags().BoolVar(&noLLM, "no-llm", false, "disable LLM features even when Claude CLI is available")
	scanCmd.Flags().BoolVar(&enableC7, "enable-c7", false, "enable C7 agent evaluation using Claude Code CLI (requires claude CLI installed)")
	scanCmd.Flags().StringVar(&outputHTML, "output-html", "", "generate self-contained HTML report at specified path")
	scanCmd.Flags().StringVar(&baselinePath, "baseline", "", "path to previous JSON output for trend comparison")
	scanCmd.Flags().BoolVar(&badgeOutput, "badge", false, "generate shields.io badge markdown URL")
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
