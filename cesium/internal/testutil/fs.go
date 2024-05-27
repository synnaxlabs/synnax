package testutil

import (
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
	"os"
)

type FSFactory func() (xfs.FS, func() error)

var FileSystems = map[string]FSFactory{
	"memFS": func() (xfs.FS, func() error) {
		return MustSucceed(xfs.NewMem().Sub("testdata")), func() error { return nil }
	},
	"osFS": func() (xfs.FS, func() error) {
		dirName := MustSucceed(os.MkdirTemp("", "test-*"))
		return MustSucceed(xfs.Default.Sub(dirName)), func() error { return xfs.Default.Remove(dirName) }
	},
}
