package main

import "testing"

func TestResolveRootPrefersPositionalArg(t *testing.T) {
	t.Setenv("SECRETS_ROOT", "/from/env")

	if got := resolveRoot([]string{"/from/arg"}); got != "/from/arg" {
		t.Fatalf("a positional path should win, got %q", got)
	}
}

func TestResolveRootSkipsFlagsForPositionalArg(t *testing.T) {
	t.Setenv("SECRETS_ROOT", "")

	if got := resolveRoot([]string{"--version"}); got != "." {
		t.Fatalf("a flag is not a path; should fall back to cwd, got %q", got)
	}
}

func TestResolveRootFallsBackToEnv(t *testing.T) {
	t.Setenv("SECRETS_ROOT", "/from/env")

	if got := resolveRoot(nil); got != "/from/env" {
		t.Fatalf("SECRETS_ROOT should be used when no positional arg, got %q", got)
	}
}

func TestResolveRootDefaultsToCwd(t *testing.T) {
	t.Setenv("SECRETS_ROOT", "")

	if got := resolveRoot(nil); got != "." {
		t.Fatalf("should default to the current directory, got %q", got)
	}
}
