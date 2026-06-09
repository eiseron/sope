package model

import (
	"reflect"
	"testing"
	"time"

	"github.com/getsops/sops/v3/decrypt"
)

func TestCreateFileRoundTripsThroughKeyring(t *testing.T) {
	id, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	entries := []Entry{
		{Key: "DATABASE_URL", Value: "postgres://u:p@h:5432/db"},
		{Key: "TOKEN", Value: "s3cr3t"},
	}
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)

	ct, err := CreateFile([]string{id.Recipient}, entries, now)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	k := NewKeyring()
	if err := k.Unlock(id.Secret); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	got, err := k.DecryptFile(ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !reflect.DeepEqual(got, entries) {
		t.Fatalf("round-trip mismatch\n got: %#v\nwant: %#v", got, entries)
	}
}

func TestCreateFileDecryptsWithSopsLibrary(t *testing.T) {
	id, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	entries := []Entry{{Key: "ONLY", Value: "value"}}
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)

	ct, err := CreateFile([]string{id.Recipient}, entries, now)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	t.Setenv("SOPS_AGE_KEY", id.Secret)
	plain, err := decrypt.Data(ct, "dotenv")
	if err != nil {
		t.Fatalf("sops failed to decrypt the created file: %v", err)
	}
	if got := parseDotenv(plain); !reflect.DeepEqual(got, entries) {
		t.Fatalf("mismatch\n got: %#v\nwant: %#v", got, entries)
	}
}

func TestCreateFileOnlyOpensForItsRecipient(t *testing.T) {
	owner, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("owner generate: %v", err)
	}
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	ct, err := CreateFile([]string{owner.Recipient}, []Entry{{Key: "K", Value: "v"}}, now)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	k := NewKeyring()
	if err := k.Unlock(wrongIdentity); err != nil {
		t.Fatalf("unlock unrelated identity: %v", err)
	}
	if _, err := k.DecryptFile(ct); err != ErrNeedUnlock {
		t.Fatalf("expected ErrNeedUnlock for an unrelated key, got %v", err)
	}
}

func TestCreateFileRejectsNoRecipients(t *testing.T) {
	_, err := CreateFile(nil, []Entry{{Key: "K", Value: "v"}}, time.Now())
	if err == nil {
		t.Fatal("expected an error when no recipients are given")
	}
}

func TestCreateFileRejectsInvalidRecipient(t *testing.T) {
	_, err := CreateFile([]string{"not-an-age-recipient"}, []Entry{{Key: "K", Value: "v"}}, time.Now())
	if err == nil {
		t.Fatal("expected an error for a malformed recipient")
	}
}
