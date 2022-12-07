package ui

import (
	"embed"
)

var Dist embed.FS

var HaveUI bool

func init() {
	Dist = dist
}
