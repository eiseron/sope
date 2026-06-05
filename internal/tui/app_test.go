package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
