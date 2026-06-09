package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureRecipient(t *testing.T) string {
	t.Helper()
	cfg, err := os.ReadFile(filepath.Join("testdata", ".sops.yaml"))
	if err != nil {
		t.Fatalf("reading fixture config: %v", err)
	}
	for _, line := range strings.Split(string(cfg), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "age:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "age:"))
		}
	}
	t.Fatal("no recipient in fixture .sops.yaml")
	return ""
}

func newFileRootRule(t *testing.T, pathRegex string) string {
	t.Helper()
	root := t.TempDir()
	body := "creation_rules:\n  - path_regex: " + pathRegex + "\n    age: " + fixtureRecipient(t) + "\n"
	if err := os.WriteFile(filepath.Join(root, ".sops.yaml"), []byte(body), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	return root
}

func newFileRoot(t *testing.T) string {
	t.Helper()
	return newFileRootRule(t, `\.enc\.env$`)
}

func TestNewFileCreatesEncryptedFileForExistingRecipient(t *testing.T) {
	root := newFileRoot(t)
	m, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(m.files) != 0 {
		t.Fatalf("expected an empty root, found %d files", len(m.files))
	}

	m = step(t, m, key('n'))
	if m.screen != screenNewFile {
		t.Fatalf("expected new-file screen, got %v", m.screen)
	}
	m.nameInput.SetValue("api")
	m = step(t, m, enter())

	if _, err := os.Stat(filepath.Join(root, "api.enc.env")); err != nil {
		t.Fatalf("expected api.enc.env on disk: %v", err)
	}
	if len(m.files) != 1 || m.files[0].Rel != "api.enc.env" {
		t.Fatalf("new file not listed: %#v", m.files)
	}
	if m.screen != screenUnlock {
		t.Fatalf("expected unlock prompt for the new file, got %v", m.screen)
	}

	m.input.SetValue(identity(t))
	m = step(t, m, enter())
	if m.screen != screenKeys {
		t.Fatalf("expected key view after unlock, got %v", m.screen)
	}
	if len(m.entries) != 0 {
		t.Fatalf("expected the new file to start empty, got %d entries", len(m.entries))
	}
}

func TestNewFileAddsCanonicalExtensionToTypedName(t *testing.T) {
	root := newFileRoot(t)
	m, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m = step(t, m, key('n'))
	m.nameInput.SetValue("ops/db")
	m = step(t, m, enter())

	if _, err := os.Stat(filepath.Join(root, "ops", "db.enc.env")); err != nil {
		t.Fatalf("expected ops/db.enc.env created with parent dir: %v", err)
	}
}

func TestNewFileRejectsInvalidNameAndCreatesNothing(t *testing.T) {
	root := newFileRoot(t)
	m, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m = step(t, m, key('n'))
	m.nameInput.SetValue("../escape")
	m = step(t, m, enter())

	if m.screen != screenNewFile {
		t.Fatalf("expected to stay on the new-file screen after a bad name, got %v", m.screen)
	}
	if m.status == "" {
		t.Fatal("expected an error status for an invalid name")
	}
	if _, err := os.Stat(filepath.Join(root, "..", "escape.enc.env")); !os.IsNotExist(err) {
		t.Fatalf("a file was created outside the root: %v", err)
	}
}

func TestNewFileRejectsNameMatchingNoRule(t *testing.T) {
	root := newFileRootRule(t, `secrets\.enc\.env$`)
	m, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m = step(t, m, key('n'))
	m.nameInput.SetValue("random")
	m = step(t, m, enter())

	if m.screen != screenNewFile {
		t.Fatalf("expected to stay on the new-file screen, got %v", m.screen)
	}
	if m.status == "" {
		t.Fatal("expected an error when the name matches no creation rule")
	}
	if _, err := os.Stat(filepath.Join(root, "random.enc.env")); !os.IsNotExist(err) {
		t.Fatal("an unmatched file was created")
	}
}

func TestNewFileRejectsExistingFile(t *testing.T) {
	root := tempRoot(t)
	m, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	before := len(m.files)
	m = step(t, m, key('n'))
	m.nameInput.SetValue("secrets")
	m = step(t, m, enter())

	if m.screen != screenNewFile {
		t.Fatalf("expected to stay on the new-file screen, got %v", m.screen)
	}
	if m.status == "" {
		t.Fatal("expected an error for an already-existing file")
	}
	if len(m.files) != before {
		t.Fatalf("file count changed from %d to %d", before, len(m.files))
	}
}

func TestNewFileEscapeReturnsToFileList(t *testing.T) {
	root := newFileRoot(t)
	m, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m = step(t, m, key('n'))
	m.nameInput.SetValue("partial")
	m = step(t, m, esc())

	if m.screen != screenFiles {
		t.Fatalf("expected to return to the file list, got %v", m.screen)
	}
	if m.nameInput.Value() != "" {
		t.Fatalf("expected the name input to be cleared, got %q", m.nameInput.Value())
	}
}
