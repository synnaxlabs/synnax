package version

import (
	"embed"
	"io/ioutil"
)

//go:embed VERSION
var fs embed.FS

func Get() string{
	f, _ := fs.Open("VERSION")
	version_name, _ := ioutil.ReadAll(f)
	return string(version_name)
}