//go:build integration

package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/eiseron/sope/lib/model"
)

func TestE2ENewFileCreateUnlockAddSaveQuit(t *testing.T) {
	root := tempRoot(t)
	m, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	waitForOutput(t, tm, "secrets.enc.env")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	waitForOutput(t, tm, "new file")

	tm.Type("api")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForOutput(t, tm, "key for api.enc.env")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	waitForOutput(t, tm, "paste recipient")

	tm.Type(fixtureRecipient(t))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForOutput(t, tm, "unlock api.enc.env")

	tm.Type(identity(t))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForOutput(t, tm, "api.enc.env")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	waitForOutput(t, tm, "add key")

	tm.Type("TOKEN=created-via-e2e")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForOutput(t, tm, "added TOKEN")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))

	sf := findFile(t, root, "api.enc.env")
	got := map[string]string{}
	for _, e := range diskEntries(t, root, sf) {
		got[e.Key] = e.Value
	}
	if got["TOKEN"] != "created-via-e2e" {
		t.Fatalf("key added to the new file not persisted: %q", got["TOKEN"])
	}
}

func findFile(t *testing.T, root, rel string) model.SecretFile {
	t.Helper()
	files, err := model.DiscoverSecretFiles(root)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	for _, f := range files {
		if f.Rel == rel {
			return f
		}
	}
	t.Fatalf("file %q not discovered after creation", rel)
	return model.SecretFile{}
}
