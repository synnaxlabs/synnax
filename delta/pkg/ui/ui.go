package ui

import (
	"embed"
)

//go:embed dist/* dist/assets/*
var Dist embed.FS
