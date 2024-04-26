package testutil

import (
	xfs "github.com/synnaxlabs/x/io/fs"
	"os"
)

var FileSystems = func() (fileSystemFactory map[string]func() xfs.FS, cleanUp func() error) {
	var (
		defaultFS xfs.FS
		dirName   string
	)

	fileSystemFactory = make(map[string]func() xfs.FS)
	fileSystemFactory["memFS"] = func() xfs.FS { return xfs.NewMem() }

	defaultFS = xfs.Default
	dirName = MustSucceed(os.MkdirTemp("", "test-*"))

	fileSystemFactory["osFS"] = func() xfs.FS {
		return MustSucceed(defaultFS.Sub(dirName))
	}

	cleanUp = func() error {
		return defaultFS.Remove(dirName)
	}

	return
}
