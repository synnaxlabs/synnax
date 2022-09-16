//go:build noui

package ui

import "embed"

var BareHTML = []byte(`<!DOCTYPE html>
<html lang="en">
<title>Delta</title>
Delta built without UI.
`)

var dist embed.FS

func init() {
	HaveUI = false
}
