package model

import "testing"

func TestValidateKeyAcceptsValidNames(t *testing.T) {
	for _, name := range []string{"FOO", "_private", "A1_B2", "TF_VAR_token", "x"} {
		if err := ValidateKey(name); err != nil {
			t.Errorf("ValidateKey(%q) = %v, want nil", name, err)
		}
	}
}

func TestValidateKeyRejectsInvalidNames(t *testing.T) {
	for _, name := range []string{"", "1leading", "with-dash", "with space", "with.dot", "sops_reserved"} {
		if err := ValidateKey(name); err == nil {
			t.Errorf("ValidateKey(%q) = nil, want error", name)
		}
	}
}

func TestHasKey(t *testing.T) {
	entries := []Entry{{Key: "A", Value: "1"}, {Key: "B", Value: "2"}}

	if !HasKey(entries, "A") {
		t.Error("HasKey should find an existing key")
	}
	if HasKey(entries, "Z") {
		t.Error("HasKey should not find an absent key")
	}
}
