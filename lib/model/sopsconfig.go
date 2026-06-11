package model

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"filippo.io/age"
	yaml "go.yaml.in/yaml/v3"
)

type creationRuleYAML struct {
	PathRegex string `yaml:"path_regex"`
	Age       string `yaml:"age"`
}

type sopsConfigYAML struct {
	CreationRules []creationRuleYAML `yaml:"creation_rules"`
}

func CollectRecipients(root string) ([]string, error) {
	rules, err := collectRules(root)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []string
	for _, rule := range rules {
		for _, rec := range rule.recipients {
			if rec == "" || seen[rec] {
				continue
			}
			seen[rec] = true
			out = append(out, rec)
		}
	}
	sort.Strings(out)
	return out, nil
}

func FileRuleRegex(rel string) string {
	return "^" + regexp.QuoteMeta(rel) + "$"
}

func ParseRecipients(raw string) ([]string, error) {
	parts := splitRecipients(raw)
	if len(parts) == 0 {
		return nil, fmt.Errorf("no recipient provided")
	}
	for _, part := range parts {
		if _, err := age.ParseX25519Recipient(part); err != nil {
			return nil, fmt.Errorf("invalid age recipient %q", part)
		}
	}
	return parts, nil
}

func EnsureCreationRule(root, pathRegex string, recipients []string) error {
	if len(recipients) == 0 {
		return fmt.Errorf("no age recipients for creation rule")
	}
	abs := filepath.Join(root, ".sops.yaml")
	existing, err := loadCreationRules(abs)
	if err != nil {
		return err
	}
	rules := make([]creationRuleYAML, 0, len(existing)+1)
	rules = append(rules, creationRuleYAML{PathRegex: pathRegex, Age: strings.Join(recipients, ",")})
	for _, rule := range existing {
		if rule.PathRegex == pathRegex {
			continue
		}
		rules = append(rules, rule)
	}
	data, err := yaml.Marshal(sopsConfigYAML{CreationRules: rules})
	if err != nil {
		return err
	}
	return os.WriteFile(abs, data, 0o644)
}

func loadCreationRules(abs string) ([]creationRuleYAML, error) {
	data, err := os.ReadFile(abs)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg sopsConfigFile
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("refusing to overwrite malformed %s: %w", abs, err)
	}
	out := make([]creationRuleYAML, 0, len(cfg.CreationRules))
	for _, rule := range cfg.CreationRules {
		out = append(out, creationRuleYAML{PathRegex: rule.PathRegex, Age: strings.Join(rule.Age, ",")})
	}
	return out, nil
}
