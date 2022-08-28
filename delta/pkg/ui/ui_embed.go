//go:build !noui

package ui

import "embed"

//go:embed dist/* dist/assets/*
var dist embed.FS

func init() {
	HaveUI = true
}
