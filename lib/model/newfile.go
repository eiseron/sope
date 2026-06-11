package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const fileExt = ".enc.env"

type NewFilePlan struct {
	File               SecretFile
	ExistingRecipients []string
}

func PlanNewFile(root, name string) (NewFilePlan, error) {
	rel, err := cleanName(name)
	if err != nil {
		return NewFilePlan{}, err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return NewFilePlan{}, err
	}
	chosen := ensureExt(rel)
	abs := filepath.Join(absRoot, chosen)
	if !isWithin(absRoot, abs) {
		return NewFilePlan{}, fmt.Errorf("refusing to create path outside root: %s", abs)
	}
	switch _, err := os.Stat(abs); {
	case err == nil:
		return NewFilePlan{}, fmt.Errorf("file already exists: %s", chosen)
	case !os.IsNotExist(err):
		return NewFilePlan{}, err
	}
	recipients, err := CollectRecipients(absRoot)
	if err != nil {
		return NewFilePlan{}, err
	}
	return NewFilePlan{
		File:               SecretFile{Abs: abs, Rel: chosen},
		ExistingRecipients: recipients,
	}, nil
}

func WriteNewFile(root string, sf SecretFile, data []byte) error {
	if !isWithin(root, sf.Abs) {
		return fmt.Errorf("refusing to write path outside root: %s", sf.Abs)
	}
	if err := os.MkdirAll(filepath.Dir(sf.Abs), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(sf.Abs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func WriteNewSecretFile(root string, sf SecretFile, recipients []string, entries []Entry, now time.Time) error {
	ct, err := CreateFile(recipients, entries, now)
	if err != nil {
		return err
	}
	if err := WriteNewFile(root, sf, ct); err != nil {
		return err
	}
	if err := EnsureCreationRule(root, FileRuleRegex(sf.Rel), recipients); err != nil {
		_ = os.Remove(sf.Abs)
		return err
	}
	return nil
}

func cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("name must not be empty")
	}
	if strings.ContainsRune(name, 0) {
		return "", fmt.Errorf("name must not contain a null byte")
	}
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("name must be relative, not absolute")
	}
	clean := filepath.Clean(name)
	if clean == "." {
		return "", fmt.Errorf("name must reference a file, not the root")
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("name must stay within the root")
	}
	return clean, nil
}

func ensureExt(rel string) string {
	if strings.HasSuffix(rel, fileExt) {
		return rel
	}
	return rel + fileExt
}
