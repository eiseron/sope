package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eiseron/sope/lib/model"
)

func TestNewFileKeyChoiceHidesReuseWithoutExistingKeys(t *testing.T) {
	root := t.TempDir()
	m, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m = atKeyChoice(t, m, "api")

	if strings.Contains(m.View(), "reuse a key") {
		t.Fatalf("reuse should not be offered when no keys exist:\n%s", m.View())
	}
	m = step(t, m, key('r'))
	if m.screen != screenNewFileKey {
		t.Fatalf("r must be a no-op without existing keys, got %v", m.screen)
	}
}

func TestNewFileReuseEncryptsToSelectedExistingRecipient(t *testing.T) {
	root := newFileRoot(t)
	m, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m = atKeyChoice(t, m, "api")

	if !strings.Contains(m.View(), "reuse a key") {
		t.Fatalf("reuse should be offered when a key exists:\n%s", m.View())
	}
	m = step(t, m, key('r'))
	if m.screen != screenReuse {
		t.Fatalf("expected the reuse picker, got %v", m.screen)
	}
	if got := m.genPlan.ExistingRecipients; len(got) != 1 || got[0] != fixtureRecipient(t) {
		t.Fatalf("unexpected reuse options: %#v", got)
	}

	step(t, m, enter())

	sf := model.SecretFile{Abs: filepath.Join(root, "api.enc.env"), Rel: "api.enc.env"}
	ct, err := model.ReadCiphertext(root, sf)
	if err != nil {
		t.Fatalf("read ciphertext: %v", err)
	}
	k := model.NewKeyring()
	if err := k.Unlock(identity(t)); err != nil {
		t.Fatalf("unlock with the reused key's secret: %v", err)
	}
	if _, err := k.DecryptFile(ct); err != nil {
		t.Fatalf("the reused recipient could not open the new file: %v", err)
	}
}

func TestNewFileReuseEscapeCreatesNothing(t *testing.T) {
	root := newFileRoot(t)
	m, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m = atKeyChoice(t, m, "api")
	m = step(t, m, key('r'))
	m = step(t, m, esc())

	if m.screen != screenFiles {
		t.Fatalf("expected to return to the file list, got %v", m.screen)
	}
	if _, err := os.Stat(filepath.Join(root, "api.enc.env")); !os.IsNotExist(err) {
		t.Fatal("a file was created despite cancelling the reuse picker")
	}
}
