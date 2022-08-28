//go:build !noui

package ui

import "embed"

//go:embed dist/* dist/assets/*
var dist embed.FS

var BareHTML []byte

func init() {
	HaveUI = true
}
