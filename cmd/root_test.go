package cmd

import (
	"bytes"
	"testing"

	"github.com/senna-lang/logosyncx/internal/version"
)

func TestRootCommand_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	// Reset args after test
	defer func() { rootCmd.SetArgs(nil) }()

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("expected help output, got empty string")
	}
}

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})

	defer func() { rootCmd.SetArgs(nil) }()

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestVersion_NotEmpty(t *testing.T) {
	if version.Version == "" {
		t.Error("Version should not be empty")
	}
}
