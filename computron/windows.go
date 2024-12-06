//go:build windows

package computron

import "embed"

//go:generate powershell -File build-windows.ps1
//go:embed all:python_install
var embeddedPython embed.FS
