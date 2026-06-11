package model

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestFileRuleRegexAnchorsAndEscapes(t *testing.T) {
	got := FileRuleRegex("ops/api+1.enc.env")
	re, err := regexp.Compile(got)
	if err != nil {
		t.Fatalf("compile %q: %v", got, err)
	}
	if !re.MatchString("ops/api+1.enc.env") {
		t.Fatalf("%q should match the exact path", got)
	}
	if re.MatchString("ops/apiX1Xenc.env") {
		t.Fatalf("%q must not treat . and + as metacharacters", got)
	}
	if re.MatchString("other/ops/api+1.enc.env") {
		t.Fatalf("%q must be anchored at the start", got)
	}
}

func TestEnsureCreationRuleCreatesConfigWhenAbsent(t *testing.T) {
	dir := t.TempDir()

	if err := EnsureCreationRule(dir, FileRuleRegex("a.enc.env"), []string{canonicalRecipient}); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	recipients, err := CollectRecipients(dir)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if want := []string{canonicalRecipient}; !reflect.DeepEqual(recipients, want) {
		t.Fatalf("recipients\n got: %#v\nwant: %#v", recipients, want)
	}
}

func TestEnsureCreationRulePrependsBeforeExisting(t *testing.T) {
	dir := t.TempDir()
	writeSops(t, dir, canonicalSops)

	id, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if err := EnsureCreationRule(dir, FileRuleRegex("a.enc.env"), []string{id.Recipient}); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	cfg, err := os.ReadFile(filepath.Join(dir, ".sops.yaml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	specific := strings.Index(string(cfg), `^a\.enc\.env$`)
	catchAll := strings.Index(string(cfg), `path_regex: \.enc\.env$`)
	if specific < 0 || catchAll < 0 {
		t.Fatalf("both rules should remain present:\n%s", cfg)
	}
	if specific > catchAll {
		t.Fatalf("the file-specific rule must come before the catch-all:\n%s", cfg)
	}
}

func TestEnsureCreationRuleReplacesSamePathRegex(t *testing.T) {
	dir := t.TempDir()
	regex := FileRuleRegex("a.enc.env")
	id1, _ := GenerateIdentity()
	id2, _ := GenerateIdentity()

	if err := EnsureCreationRule(dir, regex, []string{id1.Recipient}); err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	if err := EnsureCreationRule(dir, regex, []string{id2.Recipient}); err != nil {
		t.Fatalf("second ensure: %v", err)
	}

	cfg, err := os.ReadFile(filepath.Join(dir, ".sops.yaml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Count(string(cfg), `^a\.enc\.env$`) != 1 {
		t.Fatalf("the rule should be replaced, not duplicated:\n%s", cfg)
	}
	if strings.Contains(string(cfg), id1.Recipient) {
		t.Fatalf("the stale recipient should be gone:\n%s", cfg)
	}
	if !strings.Contains(string(cfg), id2.Recipient) {
		t.Fatalf("the new recipient should be present:\n%s", cfg)
	}
}

func TestEnsureCreationRuleRefusesMalformedConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".sops.yaml"), []byte("creation_rules: : :\n"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := EnsureCreationRule(dir, FileRuleRegex("a.enc.env"), []string{canonicalRecipient}); err == nil {
		t.Fatal("expected a refusal to overwrite a malformed config")
	}
}

func TestParseRecipientsAcceptsValidList(t *testing.T) {
	id, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	got, err := ParseRecipients(canonicalRecipient + ", " + id.Recipient)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if want := []string{canonicalRecipient, id.Recipient}; !reflect.DeepEqual(got, want) {
		t.Fatalf("recipients\n got: %#v\nwant: %#v", got, want)
	}
}

func TestParseRecipientsRejectsInvalid(t *testing.T) {
	cases := map[string]string{
		"empty":       "",
		"not age":     "hunter2",
		"secret key":  "AGE-SECRET-KEY-1QQPQQ",
		"one bad":     canonicalRecipient + ", nope",
		"only spaces": "   ",
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseRecipients(input); err == nil {
				t.Fatalf("expected %q to be rejected", input)
			}
		})
	}
}

func TestCollectRecipientsDedupesAndSorts(t *testing.T) {
	dir := t.TempDir()
	id, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	body := "creation_rules:\n" +
		"  - path_regex: a\\.enc\\.env$\n    age: " + id.Recipient + "\n" +
		"  - path_regex: b\\.enc\\.env$\n    age: " + canonicalRecipient + "\n" +
		"  - path_regex: c\\.enc\\.env$\n    age: " + id.Recipient + "\n"
	writeSops(t, dir, body)

	got, err := CollectRecipients(dir)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	want := []string{canonicalRecipient, id.Recipient}
	sortedWant := append([]string(nil), want...)
	if id.Recipient < canonicalRecipient {
		sortedWant = []string{id.Recipient, canonicalRecipient}
	}
	if !reflect.DeepEqual(got, sortedWant) {
		t.Fatalf("recipients\n got: %#v\nwant: %#v", got, sortedWant)
	}
}
