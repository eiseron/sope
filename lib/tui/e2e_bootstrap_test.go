//go:build integration

package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestE2EBootstrapFirstFileCreatesConfigAndFile(t *testing.T) {
	root := t.TempDir()
	m, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	waitForOutput(t, tm, "no encrypted files found")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	waitForOutput(t, tm, "new file")

	tm.Type("secrets")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForOutput(t, tm, "Save this secret key")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	waitForOutput(t, tm, "secrets.enc.env")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))

	if _, err := os.Stat(filepath.Join(root, ".sops.yaml")); err != nil {
		t.Fatalf("bootstrap did not write .sops.yaml: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "secrets.enc.env")); err != nil {
		t.Fatalf("bootstrap did not write the secret file: %v", err)
	}
}
