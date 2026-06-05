package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSecretFilesMatchesCreationRule(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "example-ops")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repo, ".sops.yaml"),
		"creation_rules:\n  - path_regex: secrets\\.enc\\.env$\n    age: age1example\n")
	writeFile(t, filepath.Join(repo, "secrets.enc.env"), "KEY=ENC[...]\n")
	writeFile(t, filepath.Join(repo, "README.md"), "not a secret\n")

	files, err := DiscoverSecretFiles(root)
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected exactly one match, got %d: %#v", len(files), files)
	}
	if files[0].Rel != filepath.Join("example-ops", "secrets.enc.env") {
		t.Fatalf("unexpected match Rel: %q", files[0].Rel)
	}
}

func TestDiscoverSecretFilesFindsRealFixture(t *testing.T) {
	files, err := DiscoverSecretFiles("testdata")
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}

	var found bool
	for _, f := range files {
		if f.Rel == "secrets.enc.env" {
			found = true
		}
		if f.Rel == "keys.txt" {
			t.Fatalf("the age key file must never be discovered as a secret file")
		}
	}
	if !found {
		t.Fatalf("expected to discover secrets.enc.env, got %#v", files)
	}
}

func TestIsWithinRejectsTraversal(t *testing.T) {
	root := t.TempDir()

	cases := []struct {
		name   string
		target string
		want   bool
	}{
		{"child", filepath.Join(root, "sub", "file"), true},
		{"root itself", root, true},
		{"parent traversal", filepath.Join(root, "..", "escape"), false},
		{"absolute outside", "/etc/passwd", false},
	}
	for _, tc := range cases {
		if got := isWithin(root, tc.target); got != tc.want {
			t.Errorf("isWithin(root, %s) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestReadCiphertextRejectsPathOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secrets.enc.env")
	writeFile(t, outside, "KEY=ENC[...]\n")

	_, err := ReadCiphertext(root, SecretFile{Abs: outside, Rel: "secrets.enc.env"})

	if err == nil {
		t.Fatal("expected ReadCiphertext to refuse a path outside root")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
