//go:build integration

package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/eiseron/sope/lib/model"
)

func TestE2EShellExportsSecretsToSubprocess(t *testing.T) {
	k := model.NewKeyring()
	if err := k.Unlock(identity(t)); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	ct, err := os.ReadFile(filepath.Join("testdata", "secrets.enc.env"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	entries, err := k.DecryptFile(ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	env := shellEnv("secrets.enc.env", entries)

	cmd := exec.Command("/bin/sh", "-c", `printf %s "$TF_VAR_token"`)
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("subshell run: %v", err)
	}
	if string(out) != "s3cr3t-token-value" {
		t.Fatalf("decrypted secret not visible in the subshell env: %q", out)
	}

	cmd = exec.Command("/bin/sh", "-c", `printf %s "$SOPE_FILE"`)
	cmd.Env = env
	out, err = cmd.Output()
	if err != nil {
		t.Fatalf("subshell run: %v", err)
	}
	if string(out) != "secrets.enc.env" {
		t.Fatalf("SOPE_FILE not set in the subshell env: %q", out)
	}
}
