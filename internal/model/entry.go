package model

import (
	"bufio"
	"bytes"
	"strings"
)

const sopsMetaPrefix = "sops_"

type Entry struct {
	Key   string
	Value string
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
