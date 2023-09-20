package version

import (
	"embed"
	"io"
)

//go:embed VERSION
var fs embed.FS

func Get() string {
	f, _ := fs.Open("VERSION")
	v, _ := io.ReadAll(f)
	return string(v)
}
