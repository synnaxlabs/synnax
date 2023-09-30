package version

import (
	"embed"
	"github.com/synnaxlabs/x/git"
	"go.uber.org/zap"
	"io"
)

//go:embed VERSION
var fs embed.FS

const unknown = "unknown"
const errorMsg = "unexpected failure to resolve version"

func Prod() string {
	f, err := fs.Open("VERSION")
	if err != nil {
		zap.S().Errorw(errorMsg, "error", err)
		return unknown
	}
	v, err := io.ReadAll(f)
	if err != nil {
		zap.S().Errorw(errorMsg, "error", err)
		return unknown
	}
	return string(v)
}

func Dev() (string, error) {
	return git.CurrentCommit(true)
}

func Get() string {
	d, err := Dev()
	if err == nil {
		return d
	}
	return Prod()
}
