package tui

import (
	"slices"
	"testing"

	"github.com/eiseron/sope/internal/model"
)

func TestResolveShellPrefersSopeShell(t *testing.T) {
	t.Setenv("SOPE_SHELL", "/bin/zsh")
	t.Setenv("SHELL", "/bin/bash")

	if got := resolveShell(); got != "/bin/zsh" {
		t.Fatalf("SOPE_SHELL should win, got %q", got)
	}
}

func TestResolveShellFallsBackToShellEnv(t *testing.T) {
	t.Setenv("SOPE_SHELL", "")
	t.Setenv("SHELL", "/bin/bash")

	if got := resolveShell(); got != "/bin/bash" {
		t.Fatalf("SHELL should be used, got %q", got)
	}
}

func TestResolveShellDefaultsToSh(t *testing.T) {
	t.Setenv("SOPE_SHELL", "")
	t.Setenv("SHELL", "")

	if got := resolveShell(); got != "/bin/sh" {
		t.Fatalf("should default to /bin/sh, got %q", got)
	}
}

func TestBuildShellCmdLoadsSecretsAndInheritsEnv(t *testing.T) {
	t.Setenv("EXISTING_VAR", "keep")
	cmd := buildShellCmd("/bin/sh", "ops/secrets.enc.env", []model.Entry{{Key: "SECRET", Value: "s3cr3t"}})

	if cmd.Path != "/bin/sh" {
		t.Fatalf("unexpected shell path: %q", cmd.Path)
	}
	if !slices.Contains(cmd.Env, "SECRET=s3cr3t") {
		t.Fatal("the secret should be in the subshell environment")
	}
	if !slices.Contains(cmd.Env, "EXISTING_VAR=keep") {
		t.Fatal("the subshell should inherit the existing environment")
	}
	if !slices.Contains(cmd.Env, "SOPE_FILE=ops/secrets.enc.env") {
		t.Fatal("the subshell should know which file is loaded via SOPE_FILE")
	}
}
