//go:build linux

package computron

import "embed"

//go:generate sh build-linux.sh
//go:embed all:python_install
var embeddedPython embed.FS
