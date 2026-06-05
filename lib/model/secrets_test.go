package model

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const wrongIdentity = "AGE-SECRET-KEY-1ZTQAC48SV8D6V4AU9DJ97ZJ28VFKHA589F54QJA60G0DUW5XE2HQ8TS63Q"

func fixtureCiphertext(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "secrets.enc.env"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	return data
}

func fixtureIdentity(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "keys.txt"))
	if err != nil {
		t.Fatalf("reading fixture key: %v", err)
	}
	return string(data)
}

func TestDecryptFileNeedsUnlockBeforeAnyIdentity(t *testing.T) {
	k := NewKeyring()

	_, err := k.DecryptFile(fixtureCiphertext(t))

	if err != ErrNeedUnlock {
		t.Fatalf("expected ErrNeedUnlock with an empty keyring, got %v", err)
	}
}

func TestDecryptFileReturnsEntriesAfterUnlock(t *testing.T) {
	k := NewKeyring()
	if err := k.Unlock(fixtureIdentity(t)); err != nil {
		t.Fatalf("unlock with fixture key failed: %v", err)
	}

	got, err := k.DecryptFile(fixtureCiphertext(t))
	if err != nil {
		t.Fatalf("decrypt after unlock failed: %v", err)
	}

	want := []Entry{
		{Key: "TF_VAR_token", Value: "s3cr3t-token-value"},
		{Key: "DATABASE_URL", Value: "postgres://user:pass@host:5432/db"},
		{Key: "WITH_EQUALS", Value: "a=b=c"},
		{Key: "WITH_SPECIAL", Value: "p@ss:w/rd+="},
		{Key: "PUBLIC_HOST_unencrypted", Value: "example.com"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decrypted entries mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestUnlockRejectsMalformedKey(t *testing.T) {
	k := NewKeyring()

	err := k.Unlock("this-is-not-an-age-key")

	if err == nil {
		t.Fatal("expected an error unlocking with a malformed key")
	}
	if !k.Empty() {
		t.Fatal("a rejected key must not be added to the keyring")
	}
}

func TestWrongIdentityCannotDecrypt(t *testing.T) {
	k := NewKeyring()
	if err := k.Unlock(wrongIdentity); err != nil {
		t.Fatalf("the wrong key should still be a valid identity: %v", err)
	}

	_, err := k.DecryptFile(fixtureCiphertext(t))

	if err != ErrNeedUnlock {
		t.Fatalf("a valid but non-matching identity must not decrypt; got %v", err)
	}
}
