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

func Get() string {
	c, err := git.CurrentCommit(true)
	if err == nil {
		return c
	}
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
