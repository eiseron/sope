//go:build integration

package tui

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func waitForOutput(t *testing.T, tm *teatest.TestModel, want string) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte(want))
	}, teatest.WithDuration(5*time.Second))
}

func TestE2EUnlockRevealEditSaveQuit(t *testing.T) {
	root := tempRoot(t)
	m, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	waitForOutput(t, tm, "secrets.enc.env")

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForOutput(t, tm, "age secret key")

	tm.Type(identity(t))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForOutput(t, tm, "TF_VAR_token")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	waitForOutput(t, tm, "s3cr3t-token-value")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	waitForOutput(t, tm, "edit TF_VAR_token")

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlU})
	tm.Type("e2e-rotated")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForOutput(t, tm, "saved TF_VAR_token")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))

	entries := diskEntries(t, root, m.files[0])
	for _, e := range entries {
		if e.Key == "TF_VAR_token" && e.Value != "e2e-rotated" {
			t.Fatalf("e2e edit not persisted: %q", e.Value)
		}
	}
}
