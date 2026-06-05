package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eiseron/sope/internal/model"
)

type screen int

const (
	screenFiles screen = iota
	screenUnlock
	screenKeys
)

type Model struct {
	root    string
	keyring *model.Keyring
	files   []model.SecretFile
	fileCur int

	screen screen

	input     textinput.Model
	pending   model.SecretFile
	pendingCT []byte
	unlockErr string

	current  model.SecretFile
	entries  []model.Entry
	keyCur   int
	revealed map[int]bool

	status string
	width  int
	height int
}

func New(root string) (Model, error) {
	files, err := model.DiscoverSecretFiles(root)
	if err != nil {
		return Model{}, err
	}
	ti := textinput.New()
	ti.EchoMode = textinput.EchoPassword
	ti.Placeholder = "AGE-SECRET-KEY-1..."
	return Model{
		root:     root,
		keyring:  model.NewKeyring(),
		files:    files,
		input:    ti,
		revealed: map[int]bool{},
	}, nil
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		switch m.screen {
		case screenFiles:
			return m.updateFiles(msg)
		case screenUnlock:
			return m.updateUnlock(msg)
		case screenKeys:
			return m.updateKeys(msg)
		}
	}
	return m, nil
}

func (m Model) updateFiles(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.fileCur < len(m.files)-1 {
			m.fileCur++
		}
	case "k", "up":
		if m.fileCur > 0 {
			m.fileCur--
		}
	case "enter":
		if len(m.files) > 0 {
			return m.openFile(m.files[m.fileCur]), nil
		}
	}
	return m, nil
}

func (m Model) updateUnlock(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.input.Reset()
		m.unlockErr = ""
		m.screen = screenFiles
		return m, nil
	case "enter":
		return m.submitUnlock(), nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "h", "left":
		m.screen = screenFiles
	case "j", "down":
		if m.keyCur < len(m.entries)-1 {
			m.keyCur++
		}
	case "k", "up":
		if m.keyCur > 0 {
			m.keyCur--
		}
	case "r", " ":
		m.revealed[m.keyCur] = !m.revealed[m.keyCur]
	}
	return m, nil
}

func (m Model) openFile(sf model.SecretFile) Model {
	ct, err := model.ReadCiphertext(m.root, sf)
	if err != nil {
		m.status = "error: " + err.Error()
		return m
	}
	entries, err := m.keyring.DecryptFile(ct)
	switch {
	case errors.Is(err, model.ErrNeedUnlock):
		m.pending = sf
		m.pendingCT = ct
		m.unlockErr = ""
		m.input.Reset()
		m.input.Focus()
		m.screen = screenUnlock
	case err != nil:
		m.status = "error: " + err.Error()
	default:
		m = m.withKeys(sf, entries)
	}
	return m
}

func (m Model) submitUnlock() Model {
	if err := m.keyring.Unlock(m.input.Value()); err != nil {
		m.unlockErr = "invalid age key"
		return m
	}
	entries, err := m.keyring.DecryptFile(m.pendingCT)
	if err != nil {
		m.unlockErr = "key does not match this file"
		return m
	}
	m.input.Reset()
	m.unlockErr = ""
	return m.withKeys(m.pending, entries)
}

func (m Model) withKeys(sf model.SecretFile, entries []model.Entry) Model {
	m.current = sf
	m.entries = entries
	m.keyCur = 0
	m.revealed = map[int]bool{}
	m.status = ""
	m.screen = screenKeys
	return m
}

func (m Model) View() string {
	switch m.screen {
	case screenUnlock:
		return m.viewUnlock()
	case screenKeys:
		return m.viewKeys()
	default:
		return m.viewFiles()
	}
}

func (m Model) viewFiles() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("sope") + "\n\n")
	if len(m.files) == 0 {
		b.WriteString("no encrypted files found under " + m.root + "\n")
	}
	for i, f := range m.files {
		cursor := "  "
		if i == m.fileCur {
			cursor = cursorStyle.Render("> ")
		}
		b.WriteString(cursor + f.Rel + "\n")
	}
	if m.status != "" {
		b.WriteString("\n" + errStyle.Render(m.status) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("j/k move · enter open · q quit"))
	return b.String()
}

func (m Model) viewUnlock() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("unlock "+m.pending.Rel) + "\n\n")
	b.WriteString("age secret key:\n" + m.input.View() + "\n")
	if m.unlockErr != "" {
		b.WriteString("\n" + errStyle.Render(m.unlockErr) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("enter unlock · esc cancel"))
	return b.String()
}

func (m Model) viewKeys() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.current.Rel) + "\n\n")
	for i, e := range m.entries {
		cursor := "  "
		if i == m.keyCur {
			cursor = cursorStyle.Render("> ")
		}
		value := maskStyle.Render(mask)
		if m.revealed[i] {
			value = e.Value
		}
		b.WriteString(fmt.Sprintf("%s%s = %s\n", cursor, e.Key, value))
	}
	b.WriteString("\n" + helpStyle.Render("j/k move · r reveal · esc back · q quit"))
	return b.String()
}
