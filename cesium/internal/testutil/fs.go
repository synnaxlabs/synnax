package testutil

import (
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
	"io"
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

func ReplicateFS(srcFS, destFS xfs.FS) error {
	items, err := srcFS.List("")
	if err != nil {
		return err
	}

	for _, item := range items {
		if item.IsDir() {
			// Create directory in destination
			subDestFS, err := destFS.Sub(item.Name())
			if err != nil {
				return err
			}
			subSrcFS, err := srcFS.Sub(item.Name())
			if err != nil {
				return err
			}

			if err := ReplicateFS(subSrcFS, subDestFS); err != nil {
				return err
			}
		} else {
			// Copy file from source to destination
			srcFile, err := srcFS.Open(item.Name(), os.O_RDONLY)
			if err != nil {
				return err
			}

			destFile, err := destFS.Open(item.Name(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
			if err != nil {
				return errors.CombineErrors(err, srcFile.Close())
			}

			if _, err := io.Copy(destFile, srcFile); err != nil {
				srcErr := srcFile.Close()
				dstErr := destFile.Close()
				return errors.CombineErrors(err, errors.CombineErrors(srcErr, dstErr))
			}

			if err := destFile.Sync(); err != nil {
				srcErr := srcFile.Close()
				dstErr := destFile.Close()
				return errors.CombineErrors(err, errors.CombineErrors(srcErr, dstErr))
			}
			err = srcFile.Close()
			if err != nil {
				return errors.CombineErrors(err, destFile.Close())
			}
			err = destFile.Close()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
