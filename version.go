package main

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionFile string

func version() string {
	return strings.TrimSpace(versionFile)
}
