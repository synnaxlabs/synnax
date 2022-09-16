package ui

import (
	"embed"
)

var Dist embed.FS

var HaveUI = false

func init() {
	if HaveUI {
		Dist = dist
	}
}
