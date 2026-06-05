package model

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/getsops/sops/v3/decrypt"
)

func secretKeyLine(t *testing.T) string {
	t.Helper()
	for _, line := range strings.Split(fixtureIdentity(t), "\n") {
		if strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			return line
		}
	}
	t.Fatal("no AGE-SECRET-KEY line in fixture")
	return ""
}

func recipientLine(blob []byte) string {
	for _, line := range strings.Split(string(blob), "\n") {
		if strings.HasPrefix(line, "sops_age__list_0__map_recipient=") {
			return line
		}
	}
	return ""
}

func TestEncryptFilePreservesRecipientAndRoundTrips(t *testing.T) {
	k := NewKeyring()
	if err := k.Unlock(fixtureIdentity(t)); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	orig := fixtureCiphertext(t)
	entries, err := k.DecryptFile(orig)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	entries[0].Value = "rotated-value"
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)

	out, err := k.EncryptFile(orig, entries, now)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if got, want := recipientLine(out), recipientLine(orig); got != want {
		t.Fatalf("recipient changed\n orig: %s\n out:  %s", want, got)
	}

	t.Setenv("SOPS_AGE_KEY", secretKeyLine(t))
	plain, err := decrypt.Data(out, "dotenv")
	if err != nil {
		t.Fatalf("re-encrypted file failed to decrypt (MAC invalid?): %v", err)
	}
	if got := parseDotenv(plain); !reflect.DeepEqual(got, entries) {
		t.Fatalf("round-trip mismatch\n got: %#v\nwant: %#v", got, entries)
	}
}
