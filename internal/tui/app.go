package tui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eiseron/sope/internal/model"
)

type screen int

const (
	screenFiles screen = iota
	screenUnlock
	screenKeys
	screenEdit
	screenAdd
	screenDelete
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

	current   model.SecretFile
	currentCT []byte
	entries   []model.Entry
	keyCur    int
	revealed  map[int]bool
	editInput textinput.Model
	addInput  textinput.Model

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
	add := textinput.New()
	add.Placeholder = "KEY=value"
	return Model{
		root:      root,
		keyring:   model.NewKeyring(),
		files:     files,
		input:     ti,
		editInput: textinput.New(),
		addInput:  add,
		revealed:  map[int]bool{},
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
		case screenEdit:
			return m.updateEdit(msg)
		case screenAdd:
			return m.updateAdd(msg)
		case screenDelete:
			return m.updateDelete(msg)
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
	case "e":
		if len(m.entries) > 0 {
			m.editInput.SetValue(m.entries[m.keyCur].Value)
			m.editInput.CursorEnd()
			m.editInput.Focus()
			m.status = ""
			m.screen = screenEdit
			return m, textinput.Blink
		}
	case "a":
		m.addInput.Reset()
		m.addInput.Focus()
		m.status = ""
		m.screen = screenAdd
		return m, textinput.Blink
	case "d":
		if len(m.entries) > 0 {
			m.status = ""
			m.screen = screenDelete
		}
	}
	return m, nil
}

func (m Model) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.editInput.Reset()
		m.screen = screenKeys
		return m, nil
	case "enter":
		edited := make([]model.Entry, len(m.entries))
		copy(edited, m.entries)
		edited[m.keyCur].Value = m.editInput.Value()
		m.editInput.Reset()
		return m.persist(edited, m.keyCur, "saved "+edited[m.keyCur].Key), nil
	}
	var cmd tea.Cmd
	m.editInput, cmd = m.editInput.Update(msg)
	return m, cmd
}

func (m Model) updateAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.addInput.Reset()
		m.screen = screenKeys
		return m, nil
	case "enter":
		return m.saveAdd(), nil
	}
	var cmd tea.Cmd
	m.addInput, cmd = m.addInput.Update(msg)
	return m, cmd
}

func (m Model) updateDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "y":
		removed := m.entries[m.keyCur].Key
		next := make([]model.Entry, 0, len(m.entries)-1)
		next = append(next, m.entries[:m.keyCur]...)
		next = append(next, m.entries[m.keyCur+1:]...)
		cur := m.keyCur
		if cur >= len(next) {
			cur = len(next) - 1
		}
		if cur < 0 {
			cur = 0
		}
		return m.persist(next, cur, "deleted "+removed), nil
	default:
		m.screen = screenKeys
	}
	return m, nil
}

func (m Model) saveAdd() Model {
	raw := m.addInput.Value()
	eq := strings.IndexByte(raw, '=')
	if eq <= 0 {
		m.status = "expected KEY=value"
		return m
	}
	key, value := raw[:eq], raw[eq+1:]
	if err := model.ValidateKey(key); err != nil {
		m.status = err.Error()
		return m
	}
	if model.HasKey(m.entries, key) {
		m.status = "key already exists: " + key
		return m
	}
	next := append(append([]model.Entry{}, m.entries...), model.Entry{Key: key, Value: value})
	m.addInput.Reset()
	return m.persist(next, len(next)-1, "added "+key)
}

func (m Model) persist(entries []model.Entry, cursor int, status string) Model {
	ct, err := m.keyring.EncryptFile(m.currentCT, entries, time.Now())
	if err != nil {
		m.status = "error: " + err.Error()
		m.screen = screenKeys
		return m
	}
	if err := model.WriteCiphertext(m.root, m.current, ct); err != nil {
		m.status = "error: " + err.Error()
		m.screen = screenKeys
		return m
	}
	m.entries = entries
	m.currentCT = ct
	m.keyCur = cursor
	m.status = status
	m.screen = screenKeys
	return m
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
		m = m.withKeys(sf, ct, entries)
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
	return m.withKeys(m.pending, m.pendingCT, entries)
}

func (m Model) withKeys(sf model.SecretFile, ct []byte, entries []model.Entry) Model {
	m.current = sf
	m.currentCT = ct
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
	case screenEdit:
		return m.viewEdit()
	case screenAdd:
		return m.viewAdd()
	case screenDelete:
		return m.viewDelete()
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
	if m.status != "" {
		b.WriteString("\n" + helpStyle.Render(m.status) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("j/k move · r reveal · e edit · a add · d delete · esc back · q quit"))
	return b.String()
}

func (m Model) viewEdit() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("edit "+m.entries[m.keyCur].Key) + "\n\n")
	b.WriteString(m.editInput.View() + "\n")
	b.WriteString("\n" + helpStyle.Render("enter save · esc cancel"))
	return b.String()
}

func (m Model) viewAdd() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("add key") + "\n\n")
	b.WriteString(m.addInput.View() + "\n")
	if m.status != "" {
		b.WriteString("\n" + errStyle.Render(m.status) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("enter save · esc cancel"))
	return b.String()
}

func (m Model) viewDelete() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("delete "+m.entries[m.keyCur].Key) + "\n\n")
	b.WriteString("delete this key? ")
	b.WriteString("\n" + helpStyle.Render("y delete · any other key cancel"))
	return b.String()
}
