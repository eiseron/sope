package model

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func writeSops(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".sops.yaml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write .sops.yaml: %v", err)
	}
}

const canonicalRecipient = "age1lgutd6s2dqaze2j8w97cq44623vt5d9u9433h26988tun7yl09cqrsvnf3"

const canonicalSops = "creation_rules:\n  - path_regex: \\.enc\\.env$\n    age: " + canonicalRecipient + "\n"

func TestPlanNewFileAppendsCanonicalExtension(t *testing.T) {
	dir := t.TempDir()
	writeSops(t, dir, canonicalSops)

	plan, err := PlanNewFile(dir, "secrets")
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.File.Rel != "secrets.enc.env" {
		t.Fatalf("expected extension appended, got %q", plan.File.Rel)
	}
	if want := []string{canonicalRecipient}; !reflect.DeepEqual(plan.ExistingRecipients, want) {
		t.Fatalf("existing recipients\n got: %#v\nwant: %#v", plan.ExistingRecipients, want)
	}
}

func TestPlanNewFileKeepsNameThatAlreadyMatches(t *testing.T) {
	dir := t.TempDir()
	writeSops(t, dir, canonicalSops)

	plan, err := PlanNewFile(dir, "ops/api.enc.env")
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.File.Rel != "ops/api.enc.env" {
		t.Fatalf("expected name kept as typed, got %q", plan.File.Rel)
	}
}

func TestPlanNewFileReportsNoRecipientsWhenNoConfig(t *testing.T) {
	dir := t.TempDir()

	plan, err := PlanNewFile(dir, "secrets")
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(plan.ExistingRecipients) != 0 {
		t.Fatalf("expected no recipients without a config, got %#v", plan.ExistingRecipients)
	}
	if plan.File.Rel != "secrets.enc.env" {
		t.Fatalf("expected canonical extension, got %q", plan.File.Rel)
	}
}

func TestPlanNewFileAllowsNameMatchingNoExistingRule(t *testing.T) {
	dir := t.TempDir()
	writeSops(t, dir, "creation_rules:\n  - path_regex: secrets\\.enc\\.env$\n    age: "+canonicalRecipient+"\n")

	plan, err := PlanNewFile(dir, "unmatched")
	if err != nil {
		t.Fatalf("expected a name not matching any rule to be allowed: %v", err)
	}
	if plan.File.Rel != "unmatched.enc.env" {
		t.Fatalf("unexpected file %q", plan.File.Rel)
	}
	if want := []string{canonicalRecipient}; !reflect.DeepEqual(plan.ExistingRecipients, want) {
		t.Fatalf("existing recipients\n got: %#v\nwant: %#v", plan.ExistingRecipients, want)
	}
}

func TestPlanNewFileRejectsExistingFile(t *testing.T) {
	dir := t.TempDir()
	writeSops(t, dir, canonicalSops)
	if err := os.WriteFile(filepath.Join(dir, "secrets.enc.env"), []byte("x"), 0o600); err != nil {
		t.Fatalf("seed existing file: %v", err)
	}

	_, err := PlanNewFile(dir, "secrets")
	if err == nil {
		t.Fatal("expected an error when the target file already exists")
	}
}

func TestPlanNewFileRejectsInvalidNames(t *testing.T) {
	dir := t.TempDir()
	writeSops(t, dir, canonicalSops)

	cases := map[string]string{
		"empty":            "",
		"blank":            "   ",
		"absolute":         "/etc/passwd",
		"parent traversal": "../escape",
		"sneaky traversal": "a/../../b",
		"dot":              ".",
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := PlanNewFile(dir, input); err == nil {
				t.Fatalf("expected an error for %q", input)
			}
		})
	}
}

func TestWriteNewSecretFileCreatesOpenableFileAndSpecificRule(t *testing.T) {
	root := t.TempDir()
	id, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	sf := SecretFile{Abs: filepath.Join(root, "secrets.enc.env"), Rel: "secrets.enc.env"}

	if err := WriteNewSecretFile(root, sf, []string{id.Recipient}, nil, time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := os.ReadFile(filepath.Join(root, ".sops.yaml"))
	if err != nil {
		t.Fatalf("reading .sops.yaml: %v", err)
	}
	if want := `path_regex: ^secrets\.enc\.env$`; !strings.Contains(string(cfg), want) {
		t.Fatalf(".sops.yaml missing a file-specific rule %q:\n%s", want, cfg)
	}

	ct, err := ReadCiphertext(root, sf)
	if err != nil {
		t.Fatalf("read ciphertext: %v", err)
	}
	k := NewKeyring()
	if err := k.Unlock(id.Secret); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	entries, err := k.DecryptFile(ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected an empty file, got %d entries", len(entries))
	}
}

func TestWriteNewSecretFileCleansUpFileWhenRuleWriteFails(t *testing.T) {
	root := t.TempDir()
	const malformed = "creation_rules: : :\n"
	if err := os.WriteFile(filepath.Join(root, ".sops.yaml"), []byte(malformed), 0o644); err != nil {
		t.Fatalf("seed malformed config: %v", err)
	}
	id, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	sf := SecretFile{Abs: filepath.Join(root, "secrets.enc.env"), Rel: "secrets.enc.env"}

	if err := WriteNewSecretFile(root, sf, []string{id.Recipient}, nil, time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("expected a failure when the config is malformed")
	}
	if _, statErr := os.Stat(sf.Abs); !os.IsNotExist(statErr) {
		t.Fatalf("the secret file was left behind after a failed write: %v", statErr)
	}
}

func TestWriteNewFileRefusesToOverwrite(t *testing.T) {
	dir := t.TempDir()
	sf := SecretFile{Abs: filepath.Join(dir, "secrets.enc.env"), Rel: "secrets.enc.env"}
	if err := WriteNewFile(dir, sf, []byte("first")); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := WriteNewFile(dir, sf, []byte("second")); err == nil {
		t.Fatal("expected the second write to refuse overwriting")
	}
}

func TestWriteNewFileCreatesParentWith0600(t *testing.T) {
	dir := t.TempDir()
	sf := SecretFile{Abs: filepath.Join(dir, "ops", "api.enc.env"), Rel: "ops/api.enc.env"}
	if err := WriteNewFile(dir, sf, []byte("data")); err != nil {
		t.Fatalf("write: %v", err)
	}
	info, err := os.Stat(sf.Abs)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected mode 0600, got %v", info.Mode().Perm())
	}
}

func TestWriteNewFileRefusesOutsideRoot(t *testing.T) {
	dir := t.TempDir()
	sf := SecretFile{Abs: filepath.Join(dir, "..", "escape.enc.env"), Rel: "../escape.enc.env"}
	if err := WriteNewFile(dir, sf, []byte("data")); err == nil {
		t.Fatal("expected a refusal to write outside the root")
	}
}
