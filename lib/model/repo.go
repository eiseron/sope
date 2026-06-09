package model

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

type SecretFile struct {
	Abs string
	Rel string
}

type sopsConfigFile struct {
	CreationRules []struct {
		PathRegex string        `yaml:"path_regex"`
		Age       ageRecipients `yaml:"age"`
	} `yaml:"creation_rules"`
}

type ageRecipients []string

func (a *ageRecipients) UnmarshalYAML(value *yaml.Node) error {
	var single string
	if err := value.Decode(&single); err == nil {
		*a = splitRecipients(single)
		return nil
	}
	var list []string
	if err := value.Decode(&list); err != nil {
		return err
	}
	*a = list
	return nil
}

func splitRecipients(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == ' ' || r == '\t'
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

type creationRule struct {
	dir        string
	re         *regexp.Regexp
	recipients []string
}

func DiscoverSecretFiles(root string) ([]SecretFile, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rules, err := collectRules(root)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var files []SecretFile
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || isSopsConfig(d.Name()) {
			return nil
		}
		if seen[path] || !matchesAnyRule(rules, path) {
			return nil
		}
		seen[path] = true
		rel, _ := filepath.Rel(root, path)
		files = append(files, SecretFile{Abs: path, Rel: rel})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Rel < files[j].Rel })
	return files, nil
}

func ReadCiphertext(root string, sf SecretFile) ([]byte, error) {
	if !isWithin(root, sf.Abs) {
		return nil, fmt.Errorf("refusing to read path outside root: %s", sf.Abs)
	}
	return os.ReadFile(sf.Abs)
}

func WriteCiphertext(root string, sf SecretFile, data []byte) error {
	if !isWithin(root, sf.Abs) {
		return fmt.Errorf("refusing to write path outside root: %s", sf.Abs)
	}
	mode := os.FileMode(0o600)
	if info, err := os.Stat(sf.Abs); err == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(sf.Abs, data, mode)
}

func collectRules(root string) ([]creationRule, error) {
	var rules []creationRule
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !isSopsConfig(d.Name()) {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		var cfg sopsConfigFile
		if yaml.Unmarshal(data, &cfg) != nil {
			return nil
		}
		dir := filepath.Dir(path)
		for _, cr := range cfg.CreationRules {
			if cr.PathRegex == "" {
				continue
			}
			re, cerr := regexp.Compile(cr.PathRegex)
			if cerr != nil {
				continue
			}
			rules = append(rules, creationRule{dir: dir, re: re, recipients: cr.Age})
		}
		return nil
	})
	return rules, err
}

func matchesAnyRule(rules []creationRule, path string) bool {
	_, ok := matchRule(rules, path)
	return ok
}

func matchRule(rules []creationRule, path string) (creationRule, bool) {
	for _, r := range rules {
		rel, err := filepath.Rel(r.dir, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		if r.re.MatchString(rel) || r.re.MatchString(path) {
			return r, true
		}
	}
	return creationRule{}, false
}

func isWithin(root, target string) bool {
	absRoot, err1 := filepath.Abs(root)
	absTarget, err2 := filepath.Abs(target)
	if err1 != nil || err2 != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func isSopsConfig(name string) bool {
	return name == ".sops.yaml" || name == ".sops.yml"
}
