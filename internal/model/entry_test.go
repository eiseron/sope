package model

import (
	"reflect"
	"testing"
)

func TestParseDotenvPreservesOrderAndSplitsOnFirstEquals(t *testing.T) {
	blob := []byte("TF_VAR_token=s3cr3t-token-value\n" +
		"DATABASE_URL=postgres://user:pass@host:5432/db\n" +
		"WITH_EQUALS=a=b=c\n" +
		"WITH_SPECIAL=p@ss:w/rd+=\n" +
		"PUBLIC_HOST_unencrypted=example.com\n")

	got := parseDotenv(blob)

	want := []Entry{
		{Key: "TF_VAR_token", Value: "s3cr3t-token-value"},
		{Key: "DATABASE_URL", Value: "postgres://user:pass@host:5432/db"},
		{Key: "WITH_EQUALS", Value: "a=b=c"},
		{Key: "WITH_SPECIAL", Value: "p@ss:w/rd+="},
		{Key: "PUBLIC_HOST_unencrypted", Value: "example.com"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseDotenv mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestParseDotenvSkipsMetadataBlanksAndComments(t *testing.T) {
	blob := []byte("\n" +
		"# a comment\n" +
		"REAL_KEY=value\n" +
		"\n" +
		"sops_mac=ENC[...]\n" +
		"sops_version=3.11.0\n" +
		"sops_unencrypted_suffix=_unencrypted\n")

	got := parseDotenv(blob)

	want := []Entry{{Key: "REAL_KEY", Value: "value"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected only the real key, got %#v", got)
	}
}

func TestParseDotenvKeepsEmptyValue(t *testing.T) {
	got := parseDotenv([]byte("EMPTY=\n"))

	want := []Entry{{Key: "EMPTY", Value: ""}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected one entry with empty value, got %#v", got)
	}
}

func TestParseDotenvIgnoresLineWithoutEquals(t *testing.T) {
	got := parseDotenv([]byte("NOT_A_PAIR\nKEY=v\n"))

	if len(got) != 1 || got[0].Key != "KEY" {
		t.Fatalf("expected only KEY=v, got %#v", got)
	}
}

func TestFormatDotenvRoundTripsThroughParse(t *testing.T) {
	entries := []Entry{
		{Key: "A", Value: "1"},
		{Key: "B", Value: "x=y=z"},
		{Key: "C_unencrypted", Value: "plain"},
		{Key: "D", Value: ""},
	}

	got := parseDotenv(formatDotenv(entries))

	if !reflect.DeepEqual(got, entries) {
		t.Fatalf("round-trip changed entries\n got: %#v\nwant: %#v", got, entries)
	}
}
