package version

import (
	"embed"
	"fmt"
	"io/ioutil"
)

//go:embed VERSION
var fs embed.FS

func GetVersion() string{
	f, _ := fs.Open("VERSION")
	version_name, _ := ioutil.ReadAll(f)
	return string(version_name)
}