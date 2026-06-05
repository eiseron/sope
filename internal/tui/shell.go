package tui

import (
	"os"
	"os/exec"

	"github.com/eiseron/sope/internal/model"
)

type shellClosedMsg struct {
	err error
}

func resolveShell() string {
	if shell := os.Getenv("SOPE_SHELL"); shell != "" {
		return shell
	}
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/sh"
}

func shellEnv(file string, entries []model.Entry) []string {
	env := append(os.Environ(), model.EnvStrings(entries)...)
	return append(env, "SOPE_FILE="+file)
}

func buildShellCmd(shell, file string, entries []model.Entry) *exec.Cmd {
	banner := "sope: " + file + " loaded, exit to return"
	script := `printf '%s\n' "$1"; exec "$0" -i`
	cmd := exec.Command(shell, "-c", script, shell, banner)
	cmd.Env = shellEnv(file, entries)
	return cmd
}
