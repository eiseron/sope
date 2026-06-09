package model

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeSops(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".sops.yaml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write .sops.yaml: %v", err)
	}
}

const canonicalSops = "creation_rules:\n  - path_regex: \\.enc\\.env$\n    age: age1lgutd6s2dqaze2j8w97cq44623vt5d9u9433h26988tun7yl09cqrsvnf3\n"

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
	if plan.Bootstrap {
		t.Fatal("expected reuse, not bootstrap, when a rule already exists")
	}
	if want := []string{"age1lgutd6s2dqaze2j8w97cq44623vt5d9u9433h26988tun7yl09cqrsvnf3"}; !reflect.DeepEqual(plan.Recipients, want) {
		t.Fatalf("recipients\n got: %#v\nwant: %#v", plan.Recipients, want)
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

func TestPlanNewFileBootstrapsWhenNoRulesExist(t *testing.T) {
	dir := t.TempDir()

	plan, err := PlanNewFile(dir, "secrets")
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if !plan.Bootstrap {
		t.Fatal("expected bootstrap when no .sops.yaml exists")
	}
	if len(plan.Recipients) != 0 {
		t.Fatalf("expected no recipients before bootstrap, got %#v", plan.Recipients)
	}
	if plan.File.Rel != "secrets.enc.env" {
		t.Fatalf("expected canonical extension, got %q", plan.File.Rel)
	}
}

func TestPlanNewFileRejectsNameMatchingNoRule(t *testing.T) {
	dir := t.TempDir()
	writeSops(t, dir, "creation_rules:\n  - path_regex: secrets\\.enc\\.env$\n    age: age1lgutd6s2dqaze2j8w97cq44623vt5d9u9433h26988tun7yl09cqrsvnf3\n")

	_, err := PlanNewFile(dir, "unmatched")
	if err == nil {
		t.Fatal("expected an error for a name no creation rule matches")
	}
}

func TestPlanNewFileRejectsRuleWithoutRecipients(t *testing.T) {
	dir := t.TempDir()
	writeSops(t, dir, "creation_rules:\n  - path_regex: \\.enc\\.env$\n")

	_, err := PlanNewFile(dir, "secrets")
	if err == nil {
		t.Fatal("expected an error when the matching rule has no age recipients")
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

func TestWriteDefaultConfigIsDiscoverableAndRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	id, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	regex, err := WriteDefaultConfig(dir, id.Recipient)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}
	if regex != defaultPathRegex {
		t.Fatalf("unexpected regex %q", regex)
	}

	plan, err := PlanNewFile(dir, "secrets")
	if err != nil {
		t.Fatalf("plan after config write: %v", err)
	}
	if plan.Bootstrap {
		t.Fatal("config write should have removed the need to bootstrap")
	}
	if want := []string{id.Recipient}; !reflect.DeepEqual(plan.Recipients, want) {
		t.Fatalf("recipients\n got: %#v\nwant: %#v", plan.Recipients, want)
	}

	if _, err := WriteDefaultConfig(dir, id.Recipient); err == nil {
		t.Fatal("expected a refusal to overwrite an existing .sops.yaml")
	}
}
