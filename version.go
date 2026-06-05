package sope

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionFile string

func Version() string {
	return strings.TrimSpace(versionFile)
}
