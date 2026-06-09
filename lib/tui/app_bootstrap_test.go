package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eiseron/sope/lib/model"
)

func atBootstrap(t *testing.T, root string) Model {
	t.Helper()
	m, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m = step(t, m, key('n'))
	m.nameInput.SetValue("secrets")
	m = step(t, m, enter())
	if m.screen != screenBootstrap {
		t.Fatalf("expected bootstrap screen, got %v", m.screen)
	}
	return m
}

func TestBootstrapGeneratesKeyButWritesNothingUntilConfirmed(t *testing.T) {
	root := t.TempDir()
	m := atBootstrap(t, root)

	if !strings.HasPrefix(m.genIdentity.Secret, "AGE-SECRET-KEY-1") {
		t.Fatalf("no age secret generated: %q", m.genIdentity.Secret)
	}
	if _, err := os.Stat(filepath.Join(root, ".sops.yaml")); !os.IsNotExist(err) {
		t.Fatal(".sops.yaml was written before confirmation")
	}
	if _, err := os.Stat(filepath.Join(root, "secrets.enc.env")); !os.IsNotExist(err) {
		t.Fatal("the file was written before confirmation")
	}
}

func TestBootstrapConfirmCreatesConfigAndFileOpenableByShownSecret(t *testing.T) {
	root := t.TempDir()
	m := atBootstrap(t, root)
	secret := m.genIdentity.Secret
	recipient := m.genIdentity.Recipient

	m = step(t, m, key('y'))

	if m.screen != screenKeys {
		t.Fatalf("expected the new file to open after confirm, got %v", m.screen)
	}
	if len(m.entries) != 0 {
		t.Fatalf("expected an empty new file, got %d entries", len(m.entries))
	}

	cfg, err := os.ReadFile(filepath.Join(root, ".sops.yaml"))
	if err != nil {
		t.Fatalf("reading .sops.yaml: %v", err)
	}
	if !strings.Contains(string(cfg), recipient) {
		t.Fatalf(".sops.yaml is missing the recipient:\n%s", cfg)
	}

	sf := model.SecretFile{Abs: filepath.Join(root, "secrets.enc.env"), Rel: "secrets.enc.env"}
	ct, err := model.ReadCiphertext(root, sf)
	if err != nil {
		t.Fatalf("read ciphertext: %v", err)
	}
	k := model.NewKeyring()
	if err := k.Unlock(secret); err != nil {
		t.Fatalf("unlock with the shown secret: %v", err)
	}
	entries, err := k.DecryptFile(ct)
	if err != nil {
		t.Fatalf("the shown secret could not open the created file: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}

	if m.genIdentity.Secret != "" {
		t.Fatal("the generated secret was retained after confirmation")
	}
}

func TestBootstrapCancelWritesNothing(t *testing.T) {
	root := t.TempDir()
	m := atBootstrap(t, root)

	m = step(t, m, esc())

	if m.screen != screenFiles {
		t.Fatalf("expected to return to the file list, got %v", m.screen)
	}
	if m.genIdentity.Secret != "" {
		t.Fatal("the generated secret was retained after cancel")
	}
	if _, err := os.Stat(filepath.Join(root, ".sops.yaml")); !os.IsNotExist(err) {
		t.Fatal(".sops.yaml was written despite cancelling")
	}
	if _, err := os.Stat(filepath.Join(root, "secrets.enc.env")); !os.IsNotExist(err) {
		t.Fatal("the file was written despite cancelling")
	}
}

func TestBootstrapIgnoresStrayKeysAndKeepsSecretVisible(t *testing.T) {
	root := t.TempDir()
	m := atBootstrap(t, root)
	secret := m.genIdentity.Secret

	m = step(t, m, key('x'))

	if m.screen != screenBootstrap {
		t.Fatalf("a stray key left the bootstrap screen: %v", m.screen)
	}
	if m.genIdentity.Secret != secret {
		t.Fatal("a stray key discarded the generated secret")
	}
}
