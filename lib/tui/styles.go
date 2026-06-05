package tui

import "github.com/charmbracelet/lipgloss"

const mask = "••••••••"

var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	cursorStyle = lipgloss.NewStyle().Bold(true)
	maskStyle   = lipgloss.NewStyle().Faint(true)
	errStyle    = lipgloss.NewStyle().Bold(true)
	helpStyle   = lipgloss.NewStyle().Faint(true)
)
