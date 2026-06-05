package model

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

const sopsMetaPrefix = "sops_"

var keyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type Entry struct {
	Key   string
	Value string
}

func ValidateKey(name string) error {
	if name == "" {
		return fmt.Errorf("key must not be empty")
	}
	if strings.HasPrefix(name, sopsMetaPrefix) {
		return fmt.Errorf("key must not start with %q", sopsMetaPrefix)
	}
	if !keyPattern.MatchString(name) {
		return fmt.Errorf("key %q must match [A-Za-z_][A-Za-z0-9_]*", name)
	}
	return nil
}

func HasKey(entries []Entry, name string) bool {
	for _, e := range entries {
		if e.Key == name {
			return true
		}
	}
	return false
}

func parseDotenv(data []byte) []Entry {
	var entries []Entry
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := line[:eq]
		if strings.HasPrefix(key, sopsMetaPrefix) {
			continue
		}
		entries = append(entries, Entry{Key: key, Value: line[eq+1:]})
	}
	return entries
}

func formatDotenv(entries []Entry) []byte {
	var b bytes.Buffer
	for _, e := range entries {
		b.WriteString(e.Key)
		b.WriteByte('=')
		b.WriteString(e.Value)
		b.WriteByte('\n')
	}
	return b.Bytes()
}
