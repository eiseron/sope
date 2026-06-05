package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/eiseron/sope/internal/model"
)

const wrongIdentity = "AGE-SECRET-KEY-1ZTQAC48SV8D6V4AU9DJ97ZJ28VFKHA589F54QJA60G0DUW5XE2HQ8TS63Q"

func enter() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyEnter} }
func esc() tea.KeyMsg   { return tea.KeyMsg{Type: tea.KeyEsc} }
func key(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func step(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	next, _ := m.Update(msg)
	return next.(Model)
}

func newModel(t *testing.T) Model {
	t.Helper()
	m, err := New("testdata")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	return m
}

func tempRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{".sops.yaml", "secrets.enc.env", "secrets2.enc.env"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatalf("reading fixture %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(root, name), data, 0o644); err != nil {
			t.Fatalf("writing fixture %s: %v", name, err)
		}
	}
	return root
}

func unlockedAt(t *testing.T, root string) Model {
	t.Helper()
	m, err := New(root)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	m = step(t, m, enter())
	m.input.SetValue(identity(t))
	m = step(t, m, enter())
	if m.screen != screenKeys {
		t.Fatalf("setup: expected key view after unlock, got %v", m.screen)
	}
	return m
}

func identity(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "keys.txt"))
	if err != nil {
		t.Fatalf("reading fixture key: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			return line
		}
	}
	t.Fatal("no AGE-SECRET-KEY line in fixture")
	return ""
}

func TestNewListsDiscoveredFiles(t *testing.T) {
	m := newModel(t)

	if len(m.files) != 2 {
		t.Fatalf("expected 2 discovered files, got %d: %#v", len(m.files), m.files)
	}
	if m.screen != screenFiles {
		t.Fatalf("expected to start on the file list, got screen %v", m.screen)
	}
}

func TestOpeningFileWithoutIdentityPromptsUnlock(t *testing.T) {
	m := newModel(t)

	m = step(t, m, enter())

	if m.screen != screenUnlock {
		t.Fatalf("expected the unlock prompt, got screen %v", m.screen)
	}
}

func TestUnlockWithWrongKeyStaysOnPromptWithError(t *testing.T) {
	m := newModel(t)
	m = step(t, m, enter())
	m.input.SetValue(wrongIdentity)

	m = step(t, m, enter())

	if m.screen != screenUnlock {
		t.Fatalf("a non-matching key must keep the unlock prompt, got screen %v", m.screen)
	}
	if m.unlockErr == "" {
		t.Fatal("expected an error message after a non-matching key")
	}
}

func TestUnlockWithCorrectKeyMasksThenReveals(t *testing.T) {
	m := newModel(t)
	m = step(t, m, enter())
	m.input.SetValue(identity(t))
	m = step(t, m, enter())

	if m.screen != screenKeys {
		t.Fatalf("expected the key view after a correct unlock, got screen %v", m.screen)
	}

	masked := m.View()
	if !strings.Contains(masked, "TF_VAR_token") {
		t.Fatal("key name should be visible in the masked view")
	}
	if strings.Contains(masked, "s3cr3t-token-value") {
		t.Fatal("plaintext value must not appear while masked")
	}

	revealed := step(t, m, key('r')).View()
	if !strings.Contains(revealed, "s3cr3t-token-value") {
		t.Fatal("revealing the selected entry should show its plaintext value")
	}
}

func TestSecondFileWithSameRecipientSkipsUnlock(t *testing.T) {
	m := newModel(t)

	m = step(t, m, enter())
	m.input.SetValue(identity(t))
	m = step(t, m, enter())
	if m.screen != screenKeys {
		t.Fatalf("setup: expected key view, got %v", m.screen)
	}

	m = step(t, m, esc())
	m = step(t, m, key('j'))
	m = step(t, m, enter())

	if m.screen != screenKeys {
		t.Fatalf("a held identity should open the second file without a prompt, got screen %v", m.screen)
	}
	if m.current.Rel != m.files[1].Rel {
		t.Fatalf("expected to be viewing the second file %q, got %q", m.files[1].Rel, m.current.Rel)
	}
}

func TestEditSavesValueAndPersistsToDisk(t *testing.T) {
	root := tempRoot(t)
	m := unlockedAt(t, root)
	editedKey := m.entries[0].Key

	m = step(t, m, key('e'))
	if m.screen != screenEdit {
		t.Fatalf("'e' should open the edit prompt, got screen %v", m.screen)
	}
	m.editInput.SetValue("rotated")
	m = step(t, m, enter())

	if m.screen != screenKeys {
		t.Fatalf("saving should return to the key view, got screen %v", m.screen)
	}
	if m.entries[0].Value != "rotated" {
		t.Fatalf("in-memory value not updated: %q", m.entries[0].Value)
	}

	fresh := model.NewKeyring()
	if err := fresh.Unlock(identity(t)); err != nil {
		t.Fatalf("unlock fresh keyring: %v", err)
	}
	ct, err := model.ReadCiphertext(root, m.files[0])
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	entries, err := fresh.DecryptFile(ct)
	if err != nil {
		t.Fatalf("decrypt saved file: %v", err)
	}
	if entries[0].Key != editedKey || entries[0].Value != "rotated" {
		t.Fatalf("disk does not reflect the edit: %#v", entries[0])
	}
}

func TestEditCancelLeavesValueUnchanged(t *testing.T) {
	root := tempRoot(t)
	m := unlockedAt(t, root)
	original := m.entries[0].Value

	m = step(t, m, key('e'))
	m.editInput.SetValue("should-not-stick")
	m = step(t, m, esc())

	if m.screen != screenKeys {
		t.Fatalf("esc should return to the key view, got screen %v", m.screen)
	}
	if m.entries[0].Value != original {
		t.Fatalf("cancel must not change the value: got %q, want %q", m.entries[0].Value, original)
	}
}

func diskEntries(t *testing.T, root string, sf model.SecretFile) []model.Entry {
	t.Helper()
	k := model.NewKeyring()
	if err := k.Unlock(identity(t)); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	ct, err := model.ReadCiphertext(root, sf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	entries, err := k.DecryptFile(ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	return entries
}

func TestAddKeyPersistsToDisk(t *testing.T) {
	root := tempRoot(t)
	m := unlockedAt(t, root)

	m = step(t, m, key('a'))
	if m.screen != screenAdd {
		t.Fatalf("'a' should open the add prompt, got screen %v", m.screen)
	}
	m.addInput.SetValue("NEW_KEY=fresh-value")
	m = step(t, m, enter())

	if m.screen != screenKeys {
		t.Fatalf("saving an added key should return to the key view, got screen %v", m.screen)
	}
	if !model.HasKey(m.entries, "NEW_KEY") {
		t.Fatalf("added key not present in memory: %#v", m.entries)
	}

	entries := diskEntries(t, root, m.files[0])
	if !model.HasKey(entries, "NEW_KEY") {
		t.Fatalf("added key not persisted to disk: %#v", entries)
	}
}

func TestAddRejectsInvalidKey(t *testing.T) {
	root := tempRoot(t)
	m := unlockedAt(t, root)
	before := len(m.entries)

	m = step(t, m, key('a'))
	m.addInput.SetValue("1INVALID=x")
	m = step(t, m, enter())

	if m.screen != screenAdd {
		t.Fatalf("an invalid key must keep the add prompt, got screen %v", m.screen)
	}
	if m.status == "" {
		t.Fatal("expected an error status for an invalid key")
	}
	if len(m.entries) != before {
		t.Fatalf("entry count changed on a rejected add: %d -> %d", before, len(m.entries))
	}
}

func TestAddRejectsDuplicateKey(t *testing.T) {
	root := tempRoot(t)
	m := unlockedAt(t, root)
	existing := m.entries[0].Key

	m = step(t, m, key('a'))
	m.addInput.SetValue(existing + "=other")
	m = step(t, m, enter())

	if m.screen != screenAdd {
		t.Fatalf("a duplicate key must keep the add prompt, got screen %v", m.screen)
	}
	if m.status == "" {
		t.Fatal("expected an error status for a duplicate key")
	}
}

func TestDeleteKeyPersistsToDisk(t *testing.T) {
	root := tempRoot(t)
	m := unlockedAt(t, root)
	removed := m.entries[0].Key
	before := len(m.entries)

	m = step(t, m, key('d'))
	if m.screen != screenDelete {
		t.Fatalf("'d' should open the delete confirm, got screen %v", m.screen)
	}
	m = step(t, m, key('y'))

	if m.screen != screenKeys {
		t.Fatalf("confirming delete should return to the key view, got screen %v", m.screen)
	}
	if len(m.entries) != before-1 || model.HasKey(m.entries, removed) {
		t.Fatalf("key not removed in memory: %#v", m.entries)
	}

	entries := diskEntries(t, root, m.files[0])
	if model.HasKey(entries, removed) {
		t.Fatalf("deleted key still on disk: %#v", entries)
	}
}

func TestDeleteCancelKeepsKey(t *testing.T) {
	root := tempRoot(t)
	m := unlockedAt(t, root)
	before := len(m.entries)

	m = step(t, m, key('d'))
	m = step(t, m, key('n'))

	if m.screen != screenKeys {
		t.Fatalf("a non-y key should cancel and return to keys, got screen %v", m.screen)
	}
	if len(m.entries) != before {
		t.Fatalf("cancel must not delete: %d -> %d", before, len(m.entries))
	}
}
