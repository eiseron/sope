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

func TestIntegrationCreateFileReadableBySopsCLI(t *testing.T) {
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "keys.txt")
	run(t, dir, nil, "age-keygen", "-o", keyFile)
	recipient := strings.TrimSpace(run(t, dir, nil, "age-keygen", "-y", keyFile))

	entries := []Entry{
		{Key: "DATABASE_URL", Value: "postgres://u:p@h:5432/db"},
		{Key: "TOKEN", Value: "born-in-sope"},
	}
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	ct, err := CreateFile([]string{recipient}, entries, now)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	encPath := filepath.Join(dir, "secrets.enc.env")
	if err := os.WriteFile(encPath, ct, 0o600); err != nil {
		t.Fatal(err)
	}

	ageEnv := []string{"SOPS_AGE_KEY_FILE=" + keyFile}
	decrypted := run(t, dir, ageEnv, "sops", "-d", "--input-type", "dotenv", "--output-type", "dotenv", encPath)
	got := map[string]string{}
	for _, e := range parseDotenv([]byte(decrypted)) {
		got[e.Key] = e.Value
	}

	if got["TOKEN"] != "born-in-sope" {
		t.Errorf("created value not seen by the sops CLI: %q", got["TOKEN"])
	}
	if got["DATABASE_URL"] != "postgres://u:p@h:5432/db" {
		t.Errorf("created value not seen by the sops CLI: %q", got["DATABASE_URL"])
	}
}

func TestIntegrationPerFileKeysDoNotCrossDecrypt(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)

	idA, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("generate A: %v", err)
	}
	idB, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("generate B: %v", err)
	}

	sfA := SecretFile{Abs: filepath.Join(dir, "a.enc.env"), Rel: "a.enc.env"}
	sfB := SecretFile{Abs: filepath.Join(dir, "b.enc.env"), Rel: "b.enc.env"}
	if err := WriteNewSecretFile(dir, sfA, []string{idA.Recipient}, []Entry{{Key: "WHO", Value: "a"}}, now); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := WriteNewSecretFile(dir, sfB, []string{idB.Recipient}, []Entry{{Key: "WHO", Value: "b"}}, now); err != nil {
		t.Fatalf("write b: %v", err)
	}

	cfg, err := os.ReadFile(filepath.Join(dir, ".sops.yaml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	first := strings.Index(string(cfg), `^b\.enc\.env$`)
	second := strings.Index(string(cfg), `^a\.enc\.env$`)
	if first < 0 || second < 0 || first > second {
		t.Fatalf("expected the most recent rule prepended, got:\n%s", cfg)
	}

	ctA, err := ReadCiphertext(dir, sfA)
	if err != nil {
		t.Fatal(err)
	}
	kB := NewKeyring()
	if err := kB.Unlock(idB.Secret); err != nil {
		t.Fatalf("unlock B: %v", err)
	}
	if _, err := kB.DecryptFile(ctA); err == nil {
		t.Fatal("file a must not be decryptable with key b")
	}

	keyFileA := filepath.Join(dir, "a.key")
	if err := os.WriteFile(keyFileA, []byte(idA.Secret+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	decrypted := run(t, dir, []string{"SOPS_AGE_KEY_FILE=" + keyFileA}, "sops", "-d", "--input-type", "dotenv", "--output-type", "dotenv", sfA.Abs)
	got := map[string]string{}
	for _, e := range parseDotenv([]byte(decrypted)) {
		got[e.Key] = e.Value
	}
	if got["WHO"] != "a" {
		t.Fatalf("the sops CLI could not read file a with its own key: %q", got["WHO"])
	}
}
