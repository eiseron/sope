//go:build integration

package model

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func run(t *testing.T, dir string, env []string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
	return string(out)
}

func TestIntegrationSopsCLIRoundTrip(t *testing.T) {
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "keys.txt")
	run(t, dir, nil, "age-keygen", "-o", keyFile)
	recipient := strings.TrimSpace(run(t, dir, nil, "age-keygen", "-y", keyFile))

	sopsYAML := "creation_rules:\n  - path_regex: secrets\\.enc\\.env$\n    age: " + recipient + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".sops.yaml"), []byte(sopsYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	encPath := filepath.Join(dir, "secrets.enc.env")
	plain := "TF_VAR_token=initial\nDATABASE_URL=postgres://u:p@h:5432/db\nOLD_KEY=remove-me\n"
	if err := os.WriteFile(encPath, []byte(plain), 0o644); err != nil {
		t.Fatal(err)
	}

	ageEnv := []string{"SOPS_AGE_KEY_FILE=" + keyFile}
	run(t, dir, ageEnv, "sops", "-e", "-i", "--input-type", "dotenv", "--output-type", "dotenv", encPath)

	cipher, err := os.ReadFile(encPath)
	if err != nil {
		t.Fatal(err)
	}
	secret, err := os.ReadFile(keyFile)
	if err != nil {
		t.Fatal(err)
	}

	k := NewKeyring()
	if err := k.Unlock(string(secret)); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	entries, err := k.DecryptFile(cipher)
	if err != nil {
		t.Fatalf("sope decrypt: %v", err)
	}

	var edited []Entry
	for _, e := range entries {
		if e.Key == "OLD_KEY" {
			continue
		}
		if e.Key == "TF_VAR_token" {
			e.Value = "rotated-by-sope"
		}
		edited = append(edited, e)
	}
	edited = append(edited, Entry{Key: "NEW_KEY", Value: "added-by-sope"})

	out, err := k.EncryptFile(cipher, edited, time.Now())
	if err != nil {
		t.Fatalf("sope encrypt: %v", err)
	}
	if err := os.WriteFile(encPath, out, 0o644); err != nil {
		t.Fatal(err)
	}

	decrypted := run(t, dir, ageEnv, "sops", "-d", "--input-type", "dotenv", "--output-type", "dotenv", encPath)
	got := map[string]string{}
	for _, e := range parseDotenv([]byte(decrypted)) {
		got[e.Key] = e.Value
	}

	if got["TF_VAR_token"] != "rotated-by-sope" {
		t.Errorf("edited value not seen by the sops CLI: %q", got["TF_VAR_token"])
	}
	if got["NEW_KEY"] != "added-by-sope" {
		t.Errorf("added key not seen by the sops CLI: %q", got["NEW_KEY"])
	}
	if _, ok := got["OLD_KEY"]; ok {
		t.Error("removed key still present according to the sops CLI")
	}
	if got["DATABASE_URL"] != "postgres://u:p@h:5432/db" {
		t.Errorf("untouched value changed: %q", got["DATABASE_URL"])
	}
}
