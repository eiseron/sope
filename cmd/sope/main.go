package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/getsops/sops/v3/logging"
	"github.com/sirupsen/logrus"

	"github.com/eiseron/sope"
	"github.com/eiseron/sope/internal/tui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-v", "--version", "version":
			fmt.Println(sope.Version())
			return
		}
	}

	logging.SetLevel(logrus.PanicLevel)

	root := resolveRoot(os.Args[1:])

	m, err := tui.New(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func resolveRoot(args []string) string {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			return arg
		}
	}
	if env := os.Getenv("SECRETS_ROOT"); env != "" {
		return env
	}
	return "."
}
