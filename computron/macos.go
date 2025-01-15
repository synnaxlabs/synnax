//go:build darwin

package computron

import "embed"

//go:generate sh build-macos.sh
//go:embed all:python_install
var embeddedPython embed.FS
