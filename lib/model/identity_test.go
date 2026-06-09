package model

import (
	"strings"
	"testing"
)

func TestGenerateIdentityProducesUsableKeyPair(t *testing.T) {
	id, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.HasPrefix(id.Secret, "AGE-SECRET-KEY-1") {
		t.Fatalf("secret is not an age secret key: %q", id.Secret)
	}
	if !strings.HasPrefix(id.Recipient, "age1") {
		t.Fatalf("recipient is not an age recipient: %q", id.Recipient)
	}

	k := NewKeyring()
	if err := k.Unlock(id.Secret); err != nil {
		t.Fatalf("generated secret was rejected by the keyring: %v", err)
	}
}

func TestGenerateIdentityIsUniquePerCall(t *testing.T) {
	first, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("first generate: %v", err)
	}
	second, err := GenerateIdentity()
	if err != nil {
		t.Fatalf("second generate: %v", err)
	}
	if first.Secret == second.Secret || first.Recipient == second.Recipient {
		t.Fatal("two generated identities collided")
	}
}
