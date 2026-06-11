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

func atKeyChoice(t *testing.T, m Model, name string) Model {
	t.Helper()
	m = step(t, m, key('n'))
	if m.screen != screenNewFile {
		t.Fatalf("expected new-file screen, got %v", m.screen)
	}
	m.nameInput.SetValue(name)
	m = step(t, m, enter())
	if m.screen != screenNewFileKey {
		t.Fatalf("expected the key-choice screen, got %v", m.screen)
	}
	return m
}

func TestNewFilePasteCreatesEncryptedFileForRecipient(t *testing.T) {
	root := t.TempDir()
	m, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	m = atKeyChoice(t, m, "api")
	m = step(t, m, key('p'))
	if m.screen != screenPaste {
		t.Fatalf("expected the paste screen, got %v", m.screen)
	}
	m.recipInput.SetValue(fixtureRecipient(t))
	m = step(t, m, enter())

	if _, err := os.Stat(filepath.Join(root, "api.enc.env")); err != nil {
		t.Fatalf("expected api.enc.env on disk: %v", err)
	}
	cfg, err := os.ReadFile(filepath.Join(root, ".sops.yaml"))
	if err != nil {
		t.Fatalf("reading .sops.yaml: %v", err)
	}
	if !strings.Contains(string(cfg), `^api\.enc\.env$`) {
		t.Fatalf(".sops.yaml missing a file-specific rule:\n%s", cfg)
	}
	if m.screen != screenUnlock {
		t.Fatalf("expected an unlock prompt for the pasted recipient, got %v", m.screen)
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

func TestNewFilePasteRejectsInvalidRecipient(t *testing.T) {
	root := t.TempDir()
	m, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	m = atKeyChoice(t, m, "api")
	m = step(t, m, key('p'))
	m.recipInput.SetValue("not-a-real-recipient")
	m = step(t, m, enter())

	if m.screen != screenPaste {
		t.Fatalf("expected to stay on the paste screen, got %v", m.screen)
	}
	if m.status == "" {
		t.Fatal("expected an error status for an invalid recipient")
	}
	if _, err := os.Stat(filepath.Join(root, "api.enc.env")); !os.IsNotExist(err) {
		t.Fatal("a file was created from an invalid recipient")
	}
}

func TestNewFileGenerateAddsCanonicalExtensionToTypedName(t *testing.T) {
	root := t.TempDir()
	m, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m = atKeyChoice(t, m, "ops/db")
	m = step(t, m, key('g'))
	if m.screen != screenGenerate {
		t.Fatalf("expected the generate screen, got %v", m.screen)
	}
	step(t, m, key('y'))

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

func TestNewFileAllowsNameMatchingNoExistingRule(t *testing.T) {
	root := newFileRootRule(t, `secrets\.enc\.env$`)
	m, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m = atKeyChoice(t, m, "random")
	m = step(t, m, key('g'))
	step(t, m, key('y'))

	if _, err := os.Stat(filepath.Join(root, "random.enc.env")); err != nil {
		t.Fatalf("expected random.enc.env to be created for a non-matching name: %v", err)
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

func TestNewFileKeyChoiceEscapeReturnsToFileList(t *testing.T) {
	root := newFileRoot(t)
	m, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m = atKeyChoice(t, m, "api")
	m = step(t, m, esc())

	if m.screen != screenFiles {
		t.Fatalf("expected to return to the file list, got %v", m.screen)
	}
	if m.nameInput.Value() != "" {
		t.Fatalf("expected the name input to be cleared, got %q", m.nameInput.Value())
	}
	if _, err := os.Stat(filepath.Join(root, "api.enc.env")); !os.IsNotExist(err) {
		t.Fatal("a file was created despite cancelling at the key choice")
	}
}
