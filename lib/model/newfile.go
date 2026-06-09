package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	fileExt          = ".enc.env"
	defaultPathRegex = `\.enc\.env$`
)

type NewFilePlan struct {
	File       SecretFile
	Recipients []string
	Bootstrap  bool
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
	rules, err := collectRules(absRoot)
	if err != nil {
		return NewFilePlan{}, err
	}

	var chosen string
	var recipients []string
	bootstrap := false

	if len(rules) == 0 {
		chosen = ensureExt(rel)
		bootstrap = true
	} else {
		rule, candidate, ok := matchAgainstRules(rules, absRoot, rel)
		if !ok {
			return NewFilePlan{}, fmt.Errorf("name %q matches no creation rule (expected %s)", rel, ruleRegexes(rules))
		}
		if len(rule.recipients) == 0 {
			return NewFilePlan{}, fmt.Errorf("creation rule matching %q has no age recipients", candidate)
		}
		chosen = candidate
		recipients = rule.recipients
	}

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

	return NewFilePlan{
		File:       SecretFile{Abs: abs, Rel: chosen},
		Recipients: recipients,
		Bootstrap:  bootstrap,
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

func WriteDefaultConfig(root, recipient string) (string, error) {
	abs := filepath.Join(root, ".sops.yaml")
	f, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	content := fmt.Sprintf("creation_rules:\n  - path_regex: %s\n    age: %s\n", defaultPathRegex, recipient)
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return defaultPathRegex, nil
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

func matchAgainstRules(rules []creationRule, root, rel string) (creationRule, string, bool) {
	for _, candidate := range candidateNames(rel) {
		if rule, ok := matchRule(rules, filepath.Join(root, candidate)); ok {
			return rule, candidate, true
		}
	}
	return creationRule{}, "", false
}

func candidateNames(rel string) []string {
	withExt := ensureExt(rel)
	if withExt == rel {
		return []string{rel}
	}
	return []string{rel, withExt}
}

func ruleRegexes(rules []creationRule) string {
	out := make([]string, 0, len(rules))
	for _, r := range rules {
		out = append(out, r.re.String())
	}
	return strings.Join(out, ", ")
}
