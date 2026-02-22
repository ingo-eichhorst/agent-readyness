package cmd

import (
	"bytes"
	"testing"
)

func TestRootCommandHasExpectedSubcommands(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "scan" {
			found = true
			break
		}
	}
	if !found {
		t.Error("root command should have 'scan' subcommand")
	}
}

func TestRootCommandMetadata(t *testing.T) {
	if rootCmd.Use != "ars" {
		t.Errorf("expected Use='ars', got %q", rootCmd.Use)
	}
	if rootCmd.Short == "" {
		t.Error("root command should have a short description")
	}
	if rootCmd.Version == "" {
		t.Error("root command should have a version set")
	}
}

func TestVerboseFlag(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("verbose")
	if f == nil {
		t.Fatal("verbose flag not registered")
	}
	if f.Shorthand != "v" {
		t.Errorf("verbose shorthand should be 'v', got %q", f.Shorthand)
	}
	if f.DefValue != "false" {
		t.Errorf("verbose default should be 'false', got %q", f.DefValue)
	}
}

func TestSilenceErrors(t *testing.T) {
	if !rootCmd.SilenceErrors {
		t.Error("root command should have SilenceErrors=true")
	}
}

func TestExecute_HelpDoesNotPanic(t *testing.T) {
	// Execute with --help to exercise the Execute path without os.Exit
	rootCmd.SetArgs([]string{"--help"})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})
	// rootCmd.Execute() returns nil for --help; we just ensure no panic
	_ = rootCmd.Execute()
}
