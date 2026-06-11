package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eiseron/sope/lib/model"
)

func atGenerate(t *testing.T, root string) Model {
	t.Helper()
	m, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m = atKeyChoice(t, m, "secrets")
	m = step(t, m, key('g'))
	if m.screen != screenGenerate {
		t.Fatalf("expected the generate screen, got %v", m.screen)
	}
	return m
}

func TestGenerateShowsKeyButWritesNothingUntilConfirmed(t *testing.T) {
	root := t.TempDir()
	m := atGenerate(t, root)

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

func TestGenerateConfirmCreatesSpecificRuleAndFileOpenableByShownSecret(t *testing.T) {
	root := t.TempDir()
	m := atGenerate(t, root)
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
	if !strings.Contains(string(cfg), `^secrets\.enc\.env$`) {
		t.Fatalf(".sops.yaml is missing the file-specific rule:\n%s", cfg)
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

func TestGenerateDistinctKeysPerFileDoNotCrossDecrypt(t *testing.T) {
	root := t.TempDir()

	first := atGenerate(t, root)
	firstSecret := first.genIdentity.Secret
	step(t, first, key('y'))

	second, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	second = atKeyChoice(t, second, "other")
	second = step(t, second, key('g'))
	secondRecipient := second.genIdentity.Recipient
	step(t, second, key('y'))

	if firstSecret == "" || secondRecipient == "" {
		t.Fatal("expected two distinct generated identities")
	}

	sf := model.SecretFile{Abs: filepath.Join(root, "other.enc.env"), Rel: "other.enc.env"}
	ct, err := model.ReadCiphertext(root, sf)
	if err != nil {
		t.Fatalf("read ciphertext: %v", err)
	}
	k := model.NewKeyring()
	if err := k.Unlock(firstSecret); err != nil {
		t.Fatalf("unlock with the first secret: %v", err)
	}
	if _, err := k.DecryptFile(ct); err == nil {
		t.Fatal("the first file's key must not decrypt the second file")
	}
}

func TestGenerateCancelWritesNothing(t *testing.T) {
	root := t.TempDir()
	m := atGenerate(t, root)

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

func TestGenerateIgnoresStrayKeysAndKeepsSecretVisible(t *testing.T) {
	root := t.TempDir()
	m := atGenerate(t, root)
	secret := m.genIdentity.Secret

	m = step(t, m, key('x'))

	if m.screen != screenGenerate {
		t.Fatalf("a stray key left the generate screen: %v", m.screen)
	}
	if m.genIdentity.Secret != secret {
		t.Fatal("a stray key discarded the generated secret")
	}
}
