package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteBootstrapCreatesConfigAndOpenableFile(t *testing.T) {
	root := t.TempDir()
	id, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	sf := SecretFile{Abs: filepath.Join(root, "secrets.enc.env"), Rel: "secrets.enc.env"}

	if err := WriteBootstrap(root, id.Recipient, sf, nil, time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	cfg, err := os.ReadFile(filepath.Join(root, ".sops.yaml"))
	if err != nil {
		t.Fatalf("reading .sops.yaml: %v", err)
	}
	if !strings.Contains(string(cfg), id.Recipient) {
		t.Fatalf(".sops.yaml missing the recipient:\n%s", cfg)
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
		t.Fatalf("decrypt bootstrapped file: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected an empty file, got %d entries", len(entries))
	}
}

func TestWriteBootstrapCleansUpFileWhenConfigWriteFails(t *testing.T) {
	root := t.TempDir()
	const preexisting = "creation_rules: []\n"
	if err := os.WriteFile(filepath.Join(root, ".sops.yaml"), []byte(preexisting), 0o644); err != nil {
		t.Fatalf("seed .sops.yaml: %v", err)
	}
	id, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	sf := SecretFile{Abs: filepath.Join(root, "secrets.enc.env"), Rel: "secrets.enc.env"}

	err = WriteBootstrap(root, id.Recipient, sf, nil, time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected an error when the config already exists")
	}

	if _, statErr := os.Stat(sf.Abs); !os.IsNotExist(statErr) {
		t.Fatalf("the secret file was left behind after a failed bootstrap: %v", statErr)
	}
	cfg, readErr := os.ReadFile(filepath.Join(root, ".sops.yaml"))
	if readErr != nil {
		t.Fatalf("reading .sops.yaml: %v", readErr)
	}
	if string(cfg) != preexisting {
		t.Fatalf("the preexisting .sops.yaml was modified:\n%s", cfg)
	}
}
